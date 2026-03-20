import warnings

from bigclaw import deprecation, orchestration, queue, runtime, scheduler, service, workflow


def test_warn_legacy_runtime_surface_emits_deprecation_warning():
    with warnings.catch_warnings(record=True) as caught:
        warnings.simplefilter("always")
        message = deprecation.warn_legacy_runtime_surface(
            "python -m bigclaw",
            "bash scripts/ops/bigclawctl",
        )

    assert "frozen for migration-only use" in message
    assert caught
    assert issubclass(caught[0].category, DeprecationWarning)
    assert "bash scripts/ops/bigclawctl" in str(caught[0].message)


def test_legacy_runtime_modules_expose_go_mainline_replacements():
    assert runtime.GO_MAINLINE_REPLACEMENT == "bigclaw-go/internal/worker/runtime.go"
    assert scheduler.GO_MAINLINE_REPLACEMENT == "bigclaw-go/internal/scheduler/scheduler.go"
    assert workflow.GO_MAINLINE_REPLACEMENT == "bigclaw-go/internal/workflow/engine.go"
    assert orchestration.GO_MAINLINE_REPLACEMENT == "bigclaw-go/internal/workflow/orchestration.go"
    assert queue.GO_MAINLINE_REPLACEMENT == "bigclaw-go/internal/queue/queue.go"
    assert "sole implementation mainline" in runtime.LEGACY_MAINLINE_STATUS


def test_legacy_service_surface_emits_go_first_warning():
    with warnings.catch_warnings(record=True) as caught:
        warnings.simplefilter("always")
        message = service.warn_legacy_service_surface()

    assert "go run ./bigclaw-go/cmd/bigclawd" in message
    assert caught
    assert issubclass(caught[0].category, DeprecationWarning)


def test_service_module_exposes_go_mainline_replacement():
    assert service.GO_MAINLINE_REPLACEMENT == "bigclaw-go/cmd/bigclawd/main.go"
    assert "sole implementation mainline" in service.LEGACY_MAINLINE_STATUS
