package schema

import (
	"github.com/segmentio/encoding/json"
)

// PartialPackageSpec is the serializable description of a Pulumi package, with definitions of types, functions, and
// resources unparsed.
type PartialPackageSpec struct {
	// Name is the unqualified name of the package (e.g. "aws", "azure", "gcp", "kubernetes", "random")
	Name string `json:"name" yaml:"name"`
	// DisplayName is the human-friendly name of the package.
	DisplayName string `json:"displayName,omitempty" yaml:"displayName,omitempty"`
	// Version is the version of the package. The version must be valid semver.
	Version string `json:"version,omitempty" yaml:"version,omitempty"`
	// Description is the description of the package.
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
	// Keywords is the list of keywords that are associated with the package, if any.
	// Some reserved keywords can be specified as well that help with categorizing the
	// package in the Pulumi registry. `category/<name>` and `kind/<type>` are the only
	// reserved keywords at this time, where `<name>` can be one of:
	// `cloud`, `database`, `infrastructure`, `monitoring`, `network`, `utility`, `vcs`
	// and `<type>` is either `native` or `component`. If the package is a bridged Terraform
	// provider, then don't include the `kind/` label.
	Keywords []string `json:"keywords,omitempty" yaml:"keywords,omitempty"`
	// Homepage is the package's homepage.
	Homepage string `json:"homepage,omitempty" yaml:"homepage,omitempty"`
	// License indicates which license is used for the package's contents.
	License string `json:"license,omitempty" yaml:"license,omitempty"`
	// Attribution allows freeform text attribution of derived work, if needed.
	Attribution string `json:"attribution,omitempty" yaml:"attribution,omitempty"`
	// Repository is the URL at which the source for the package can be found.
	Repository string `json:"repository,omitempty" yaml:"repository,omitempty"`
	// LogoURL is the URL for the package's logo, if any.
	LogoURL string `json:"logoUrl,omitempty" yaml:"logoUrl,omitempty"`
	// PluginDownloadURL is the URL to use to acquire the provider plugin binary, if any.
	PluginDownloadURL string `json:"pluginDownloadURL,omitempty" yaml:"pluginDownloadURL,omitempty"`
	// Publisher is the name of the person or organization that authored and published the package.
	Publisher string `json:"publisher,omitempty" yaml:"publisher,omitempty"`

	// Meta contains information for the importer about this package.
	Meta *MetadataSpec `json:"meta,omitempty" yaml:"meta,omitempty"`

	// A list of allowed package name in addition to the Name property.
	AllowedPackageNames []string `json:"allowedPackageNames,omitempty" yaml:"allowedPackageNames,omitempty"`

	// Config describes the set of configuration variables defined by this package.
	Config ConfigSpec `json:"config" yaml:"config"`
	// Types is a map from type token to ComplexTypeSpec that describes the set of complex types (ie. object, enum)
	// defined by this package.
	Types map[string]json.RawMessage `json:"types,omitempty" yaml:"types,omitempty"`
	// Provider describes the provider type for this package.
	Provider ResourceSpec `json:"provider" yaml:"provider"`
	// Resources is a map from type token to ResourceSpec that describes the set of resources defined by this package.
	Resources map[string]json.RawMessage `json:"resources,omitempty" yaml:"resources,omitempty"`
	// Functions is a map from token to FunctionSpec that describes the set of functions defined by this package.
	Functions map[string]json.RawMessage `json:"functions,omitempty" yaml:"functions,omitempty"`
	// Language specifies additional language-specific data about the package.
	Language map[string]json.RawMessage `json:"language,omitempty" yaml:"language,omitempty"`
}
