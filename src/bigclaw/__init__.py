import sys as _sys

from . import runtime as runtime
from .runtime import (
    GO_MAINLINE_REPLACEMENT,
    LEGACY_PYTHON_WRAPPER_NOTICE,
    LEGACY_RUNTIME_GUIDANCE,
    append_missing_flag,
    build_github_sync_args,
    build_refill_args,
    build_workspace_bootstrap_args,
    build_workspace_validate_args,
    legacy_runtime_message,
    translate_workspace_validate_args,
    warn_legacy_runtime_surface,
)


deprecation = runtime
legacy_shim = runtime

_sys.modules[__name__ + ".deprecation"] = runtime
_sys.modules[__name__ + ".legacy_shim"] = runtime

__all__ = [
    "GO_MAINLINE_REPLACEMENT",
    "LEGACY_PYTHON_WRAPPER_NOTICE",
    "LEGACY_RUNTIME_GUIDANCE",
    "append_missing_flag",
    "build_github_sync_args",
    "build_refill_args",
    "build_workspace_bootstrap_args",
    "build_workspace_validate_args",
    "deprecation",
    "legacy_runtime_message",
    "legacy_shim",
    "runtime",
    "translate_workspace_validate_args",
    "warn_legacy_runtime_surface",
]
