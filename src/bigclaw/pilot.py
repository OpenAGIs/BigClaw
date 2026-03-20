from dataclasses import dataclass, field
from typing import List


@dataclass
class PilotKPI:
    name: str
    target: float
    actual: float
    higher_is_better: bool = True

    @property
    def met(self) -> bool:
        return self.actual >= self.target if self.higher_is_better else self.actual <= self.target


@dataclass
class PilotImplementationResult:
    customer: str
    environment: str
    kpis: List[PilotKPI] = field(default_factory=list)
    production_runs: int = 0
    incidents: int = 0

    @property
    def kpi_pass_rate(self) -> float:
        if not self.kpis:
            return 0.0
        passed = len([k for k in self.kpis if k.met])
        return round((passed / len(self.kpis)) * 100, 1)

    @property
    def ready(self) -> bool:
        return self.production_runs > 0 and self.incidents == 0 and self.kpi_pass_rate >= 80.0


def render_pilot_implementation_report(result: PilotImplementationResult) -> str:
    lines = [
        "# Pilot Implementation Report",
        "",
        f"- Customer: {result.customer}",
        f"- Environment: {result.environment}",
        f"- Production Runs: {result.production_runs}",
        f"- Incidents: {result.incidents}",
        f"- KPI Pass Rate: {result.kpi_pass_rate}%",
        f"- Ready: {result.ready}",
        "",
        "## KPI Details",
        "",
    ]
    if result.kpis:
        for k in result.kpis:
            lines.append(f"- {k.name}: target={k.target} actual={k.actual} met={k.met}")
    else:
        lines.append("- none")
    return "\n".join(lines) + "\n"
