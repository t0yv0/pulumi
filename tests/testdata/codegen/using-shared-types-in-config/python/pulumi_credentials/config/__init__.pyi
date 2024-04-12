# coding=utf-8
# *** WARNING: this file was generated by test. ***
# *** Do not edit by hand unless you're certain you know what you are doing! ***

import copy
import warnings
import pulumi
import pulumi.runtime
from typing import Any, Mapping, Optional, Sequence, Union, overload
from .. import _utilities
from .. import _enums as _root_enums
from .. import outputs as _root_outputs

hash: Optional[str]
"""
The (entirely uncryptographic) hash function used to encode the "password".
"""

password: str
"""
The password. It is very secret.
"""

shared: Optional[str]

user: Optional[str]
"""
The username. Its important but not secret.
"""
