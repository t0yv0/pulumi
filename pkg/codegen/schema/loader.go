package schema

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/edsrzf/mmap-go"
	"github.com/segmentio/encoding/json"

	"github.com/blang/semver"

	"github.com/pulumi/pulumi/sdk/v3/go/common/resource/plugin"
	"github.com/pulumi/pulumi/sdk/v3/go/common/tokens"
	"github.com/pulumi/pulumi/sdk/v3/go/common/util/contract"
	"github.com/pulumi/pulumi/sdk/v3/go/common/workspace"
)

type Loader interface {
	LoadPackage(pkg string, version *semver.Version) (*Package, error)
}

type pluginLoader struct {
	m sync.RWMutex

	host    plugin.Host
	entries map[string]*Package
	files   []*os.File
	mmaps   []mmap.MMap
}

func NewPluginLoader(host plugin.Host) Loader {
	return &pluginLoader{
		host:    host,
		entries: map[string]*Package{},
	}
}

func (l *pluginLoader) getPackage(key string) (*Package, bool) {
	l.m.RLock()
	defer l.m.RUnlock()

	p, ok := l.entries[key]
	return p, ok
}

// ensurePlugin downloads and installs the specified plugin if it does not already exist.
func (l *pluginLoader) ensurePlugin(pkg string, version *semver.Version) error {
	// TODO: schema and provider versions
	// hack: Some of the hcl2 code isn't yet handling versions, so bail out if the version is nil to avoid failing
	// 		 the download. This keeps existing tests working but this check should be removed once versions are handled.
	if version == nil {
		return nil
	}

	pkgPlugin := workspace.PluginInfo{
		Kind:    workspace.ResourcePlugin,
		Name:    pkg,
		Version: version,
	}

	tryDownload := func(dst io.WriteCloser) error {
		defer dst.Close()
		tarball, expectedByteCount, err := pkgPlugin.Download()
		if err != nil {
			return err
		}
		defer tarball.Close()
		copiedByteCount, err := io.Copy(dst, tarball)
		if err != nil {
			return err
		}
		if copiedByteCount != expectedByteCount {
			return fmt.Errorf("Expected %d bytes but copied %d when downloading plugin %s",
				expectedByteCount, copiedByteCount, pkgPlugin)
		}
		return nil
	}

	tryDownloadToFile := func() (string, error) {
		file, err := ioutil.TempFile("" /* default temp dir */, "pulumi-plugin-tar")
		if err != nil {
			return "", err
		}
		err = tryDownload(file)
		if err != nil {
			err2 := os.Remove(file.Name())
			if err2 != nil {
				return "", fmt.Errorf("Error while removing tempfile: %v. Context: %w", err2, err)
			}
			return "", err
		}
		return file.Name(), nil
	}

	downloadToFileWithRetry := func() (string, error) {
		delay := 80 * time.Millisecond
		for attempt := 0; ; attempt++ {
			tempFile, err := tryDownloadToFile()
			if err == nil {
				return tempFile, nil
			}

			if err != nil && attempt >= 5 {
				return tempFile, err
			}
			time.Sleep(delay)
			delay = delay * 2
		}
	}

	if !workspace.HasPlugin(pkgPlugin) {
		tarball, err := downloadToFileWithRetry()
		if err != nil {
			return fmt.Errorf("failed to download plugin: %s: %w", pkgPlugin, err)
		}
		defer os.Remove(tarball)
		reader, err := os.Open(tarball)
		if err != nil {
			return fmt.Errorf("failed to open downloaded plugin: %s: %w", pkgPlugin, err)
		}
		if err := pkgPlugin.Install(reader, false); err != nil {
			return fmt.Errorf("failed to install plugin %s: %w", pkgPlugin, err)
		}
	}

	return nil
}

func (l *pluginLoader) LoadPackage(pkg string, version *semver.Version) (*Package, error) {
	key := pkg + "@"
	if version != nil {
		key += version.String()
	}

	if p, ok := l.getPackage(key); ok {
		return p, nil
	}

	schemaBytes, provider, err := l.loadSchemaBytes(pkg, version)
	if err != nil {
		fmt.Printf("🚀🚀🚀🚀 error loading schema bytes: %v\n", err)
		return nil, err
	}

	var spec PackageSpec
	decoder := json.NewDecoder(bytes.NewReader(schemaBytes))
	decoder.ZeroCopy()
	if err := decoder.Decode(&spec); err != nil {
		return nil, err
	}

	p, diags, err := bindSpec(spec, nil, l, false)
	if err != nil {
		return nil, err
	}
	if diags.HasErrors() {
		return nil, diags
	}
	// Insert a version into the bound schema if the package does not provide one
	if provider != nil && p.Version == nil {
		if version == nil {
			providerInfo, err := provider.GetPluginInfo()
			if err == nil {
				version = providerInfo.Version
			}
		}

		p.Version = version
	}

	l.m.Lock()
	defer l.m.Unlock()

	if p, ok := l.entries[pkg]; ok {
		return p, nil
	}
	l.entries[key] = p

	return p, nil
}

func (l *pluginLoader) loadSchemaBytes(pkg string, version *semver.Version) ([]byte, plugin.Provider, error) {
	cachedVersion := version
	if version == nil {
		cachedVersion = &semver.Version{Major: 1}
	}
	cachedDir := fmt.Sprintf("%v-%v.json", pkg, cachedVersion.String())
	cachedPath := filepath.Join("/home/friel/.pulumi/schemas", cachedDir)

	schemaBytes, ok := l.LoadCachedSchemaBytes(pkg, version, cachedPath)
	if ok {
		return schemaBytes, nil, nil
	}

	schemaBytes, provider, err := l.loadPluginSchemaBytes(pkg, version)
	if err != nil {
		return nil, nil, err
	}

	err = os.MkdirAll(cachedDir, 0755)
	if err != nil {
		fmt.Printf("💥💥💥💥💥💥 error creating dirs: %v", err)
	} else {
		err := os.WriteFile(cachedPath, schemaBytes, 0644)
		if err != nil {
			fmt.Printf("💥💥💥💥💥💥 error writing file to cache: %v", err)
		}
	}

	return schemaBytes, provider, nil

}

func (l *pluginLoader) loadPluginSchemaBytes(pkg string, version *semver.Version) ([]byte, plugin.Provider, error) {
	if err := l.ensurePlugin(pkg, version); err != nil {
		return nil, nil, err
	}

	provider, err := l.host.Provider(tokens.Package(pkg), version)
	if err != nil {
		return nil, nil, err
	}
	contract.Assert(provider != nil)

	schemaFormatVersion := 0
	schemaBytes, err := provider.GetSchema(schemaFormatVersion)
	if err != nil {
		return nil, nil, err
	}

	return schemaBytes, provider, nil
}

func (l *pluginLoader) LoadCachedSchemaBytes(pkg string, version *semver.Version, path string) ([]byte, bool) {
	schemaFile, err := os.OpenFile(path, os.O_RDONLY, 0644)
	if err != nil {
		return nil, false
	}
	schemaMmap, err := mmap.Map(schemaFile, mmap.RDONLY, 0)
	if err != nil {
		schemaFile.Close()
		return nil, false
	}

	l.files = append(l.files, schemaFile)
	l.mmaps = append(l.mmaps, schemaMmap)

	return schemaMmap, true
}
