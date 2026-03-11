from dataclasses import dataclass, field
from typing import Dict, List


VALID_VIEW_VISIBILITY = {"private", "team", "organization"}
VALID_DIGEST_CHANNELS = {"email", "slack", "webhook"}
VALID_DIGEST_CADENCES = {"hourly", "daily", "weekly"}


@dataclass(frozen=True)
class SavedViewFilter:
    field: str
    operator: str
    value: str

    def to_dict(self) -> Dict[str, str]:
        return {
            "field": self.field,
            "operator": self.operator,
            "value": self.value,
        }

    @classmethod
    def from_dict(cls, data: Dict[str, object]) -> "SavedViewFilter":
        return cls(
            field=str(data["field"]),
            operator=str(data["operator"]),
            value=str(data["value"]),
        )


@dataclass
class SavedView:
    view_id: str
    name: str
    route: str
    owner: str
    visibility: str = "private"
    filters: List[SavedViewFilter] = field(default_factory=list)
    sort_by: str = ""
    pinned: bool = False
    is_default: bool = False

    @property
    def filter_count(self) -> int:
        return len(self.filters)

    def to_dict(self) -> Dict[str, object]:
        return {
            "view_id": self.view_id,
            "name": self.name,
            "route": self.route,
            "owner": self.owner,
            "visibility": self.visibility,
            "filters": [view_filter.to_dict() for view_filter in self.filters],
            "sort_by": self.sort_by,
            "pinned": self.pinned,
            "is_default": self.is_default,
        }

    @classmethod
    def from_dict(cls, data: Dict[str, object]) -> "SavedView":
        return cls(
            view_id=str(data["view_id"]),
            name=str(data["name"]),
            route=str(data["route"]),
            owner=str(data["owner"]),
            visibility=str(data.get("visibility", "private")),
            filters=[SavedViewFilter.from_dict(item) for item in data.get("filters", [])],
            sort_by=str(data.get("sort_by", "")),
            pinned=bool(data.get("pinned", False)),
            is_default=bool(data.get("is_default", False)),
        )


@dataclass
class AlertDigestSubscription:
    subscription_id: str
    saved_view_id: str
    channel: str
    cadence: str
    recipients: List[str] = field(default_factory=list)
    include_empty_results: bool = False
    muted: bool = False

    def to_dict(self) -> Dict[str, object]:
        return {
            "subscription_id": self.subscription_id,
            "saved_view_id": self.saved_view_id,
            "channel": self.channel,
            "cadence": self.cadence,
            "recipients": list(self.recipients),
            "include_empty_results": self.include_empty_results,
            "muted": self.muted,
        }

    @classmethod
    def from_dict(cls, data: Dict[str, object]) -> "AlertDigestSubscription":
        return cls(
            subscription_id=str(data["subscription_id"]),
            saved_view_id=str(data["saved_view_id"]),
            channel=str(data["channel"]),
            cadence=str(data["cadence"]),
            recipients=[str(recipient) for recipient in data.get("recipients", [])],
            include_empty_results=bool(data.get("include_empty_results", False)),
            muted=bool(data.get("muted", False)),
        )


@dataclass
class SavedViewCatalog:
    name: str
    version: str
    views: List[SavedView] = field(default_factory=list)
    subscriptions: List[AlertDigestSubscription] = field(default_factory=list)

    @property
    def view_index(self) -> Dict[str, SavedView]:
        return {view.view_id: view for view in self.views}

    def to_dict(self) -> Dict[str, object]:
        return {
            "name": self.name,
            "version": self.version,
            "views": [view.to_dict() for view in self.views],
            "subscriptions": [subscription.to_dict() for subscription in self.subscriptions],
        }

    @classmethod
    def from_dict(cls, data: Dict[str, object]) -> "SavedViewCatalog":
        return cls(
            name=str(data["name"]),
            version=str(data["version"]),
            views=[SavedView.from_dict(item) for item in data.get("views", [])],
            subscriptions=[
                AlertDigestSubscription.from_dict(item) for item in data.get("subscriptions", [])
            ],
        )


@dataclass
class SavedViewCatalogAudit:
    catalog_name: str
    version: str
    view_count: int
    subscription_count: int
    duplicate_view_names: Dict[str, List[str]] = field(default_factory=dict)
    invalid_visibility_views: List[str] = field(default_factory=list)
    views_missing_filters: List[str] = field(default_factory=list)
    duplicate_default_views: Dict[str, List[str]] = field(default_factory=dict)
    orphan_subscriptions: List[str] = field(default_factory=list)
    subscriptions_missing_recipients: List[str] = field(default_factory=list)
    subscriptions_with_invalid_channel: List[str] = field(default_factory=list)
    subscriptions_with_invalid_cadence: List[str] = field(default_factory=list)

    @property
    def readiness_score(self) -> float:
        if self.view_count == 0:
            return 0.0
        penalties = (
            len(self.duplicate_view_names)
            + len(self.invalid_visibility_views)
            + len(self.views_missing_filters)
            + len(self.duplicate_default_views)
            + len(self.orphan_subscriptions)
            + len(self.subscriptions_missing_recipients)
            + len(self.subscriptions_with_invalid_channel)
            + len(self.subscriptions_with_invalid_cadence)
        )
        score = max(0.0, 100 - ((penalties * 100) / self.view_count))
        return round(score, 1)

    def to_dict(self) -> Dict[str, object]:
        return {
            "catalog_name": self.catalog_name,
            "version": self.version,
            "view_count": self.view_count,
            "subscription_count": self.subscription_count,
            "duplicate_view_names": {
                key: list(values) for key, values in self.duplicate_view_names.items()
            },
            "invalid_visibility_views": list(self.invalid_visibility_views),
            "views_missing_filters": list(self.views_missing_filters),
            "duplicate_default_views": {
                key: list(values) for key, values in self.duplicate_default_views.items()
            },
            "orphan_subscriptions": list(self.orphan_subscriptions),
            "subscriptions_missing_recipients": list(self.subscriptions_missing_recipients),
            "subscriptions_with_invalid_channel": list(self.subscriptions_with_invalid_channel),
            "subscriptions_with_invalid_cadence": list(self.subscriptions_with_invalid_cadence),
        }

    @classmethod
    def from_dict(cls, data: Dict[str, object]) -> "SavedViewCatalogAudit":
        return cls(
            catalog_name=str(data["catalog_name"]),
            version=str(data["version"]),
            view_count=int(data.get("view_count", 0)),
            subscription_count=int(data.get("subscription_count", 0)),
            duplicate_view_names={
                str(key): [str(value) for value in values]
                for key, values in dict(data.get("duplicate_view_names", {})).items()
            },
            invalid_visibility_views=[
                str(name) for name in data.get("invalid_visibility_views", [])
            ],
            views_missing_filters=[str(name) for name in data.get("views_missing_filters", [])],
            duplicate_default_views={
                str(key): [str(value) for value in values]
                for key, values in dict(data.get("duplicate_default_views", {})).items()
            },
            orphan_subscriptions=[str(name) for name in data.get("orphan_subscriptions", [])],
            subscriptions_missing_recipients=[
                str(name) for name in data.get("subscriptions_missing_recipients", [])
            ],
            subscriptions_with_invalid_channel=[
                str(name) for name in data.get("subscriptions_with_invalid_channel", [])
            ],
            subscriptions_with_invalid_cadence=[
                str(name) for name in data.get("subscriptions_with_invalid_cadence", [])
            ],
        )


class SavedViewLibrary:
    def audit(self, catalog: SavedViewCatalog) -> SavedViewCatalogAudit:
        duplicate_view_names: Dict[str, List[str]] = {}
        invalid_visibility_views: List[str] = []
        views_missing_filters: List[str] = []
        duplicate_default_views: Dict[str, List[str]] = {}
        orphan_subscriptions: List[str] = []
        subscriptions_missing_recipients: List[str] = []
        subscriptions_with_invalid_channel: List[str] = []
        subscriptions_with_invalid_cadence: List[str] = []
        names_by_scope: Dict[str, List[str]] = {}
        defaults_by_scope: Dict[str, List[str]] = {}

        for view in catalog.views:
            scope = f"{view.route}:{view.owner}"
            names_by_scope.setdefault(scope, []).append(view.name)
            if view.is_default:
                defaults_by_scope.setdefault(scope, []).append(view.name)
            if view.visibility not in VALID_VIEW_VISIBILITY:
                invalid_visibility_views.append(view.name)
            if not view.filters:
                views_missing_filters.append(view.name)

        for scope, names in sorted(names_by_scope.items()):
            unique_names = sorted({name for name in names if names.count(name) > 1})
            if unique_names:
                duplicate_view_names[scope] = unique_names

        for scope, names in sorted(defaults_by_scope.items()):
            if len(names) > 1:
                duplicate_default_views[scope] = sorted(names)

        view_index = catalog.view_index
        for subscription in catalog.subscriptions:
            if subscription.saved_view_id not in view_index:
                orphan_subscriptions.append(subscription.subscription_id)
            if not subscription.recipients:
                subscriptions_missing_recipients.append(subscription.subscription_id)
            if subscription.channel not in VALID_DIGEST_CHANNELS:
                subscriptions_with_invalid_channel.append(subscription.subscription_id)
            if subscription.cadence not in VALID_DIGEST_CADENCES:
                subscriptions_with_invalid_cadence.append(subscription.subscription_id)

        return SavedViewCatalogAudit(
            catalog_name=catalog.name,
            version=catalog.version,
            view_count=len(catalog.views),
            subscription_count=len(catalog.subscriptions),
            duplicate_view_names=duplicate_view_names,
            invalid_visibility_views=sorted(invalid_visibility_views),
            views_missing_filters=sorted(views_missing_filters),
            duplicate_default_views=duplicate_default_views,
            orphan_subscriptions=sorted(orphan_subscriptions),
            subscriptions_missing_recipients=sorted(subscriptions_missing_recipients),
            subscriptions_with_invalid_channel=sorted(subscriptions_with_invalid_channel),
            subscriptions_with_invalid_cadence=sorted(subscriptions_with_invalid_cadence),
        )


def render_saved_view_report(catalog: SavedViewCatalog, audit: SavedViewCatalogAudit) -> str:
    lines = [
        "# Saved Views & Alert Digests Report",
        "",
        f"- Name: {catalog.name}",
        f"- Version: {catalog.version}",
        f"- Saved Views: {audit.view_count}",
        f"- Alert Subscriptions: {audit.subscription_count}",
        f"- Readiness Score: {audit.readiness_score:.1f}",
        "",
        "## Saved Views",
        "",
    ]

    if catalog.views:
        for view in catalog.views:
            filters = ", ".join(
                f"{view_filter.field}{view_filter.operator}{view_filter.value}"
                for view_filter in view.filters
            ) or "none"
            lines.append(
                f"- {view.name}: route={view.route} owner={view.owner} visibility={view.visibility} "
                f"filters={filters} sort={view.sort_by or 'none'} pinned={view.pinned} default={view.is_default}"
            )
    else:
        lines.append("- None")

    lines.extend(["", "## Alert Digests", ""])
    if catalog.subscriptions:
        for subscription in catalog.subscriptions:
            recipients = ", ".join(subscription.recipients) or "none"
            lines.append(
                f"- {subscription.subscription_id}: view={subscription.saved_view_id} channel={subscription.channel} "
                f"cadence={subscription.cadence} recipients={recipients} "
                f"include_empty={subscription.include_empty_results} muted={subscription.muted}"
            )
    else:
        lines.append("- None")

    lines.extend(["", "## Gaps", ""])
    duplicate_names = (
        "; ".join(f"{scope}={', '.join(names)}" for scope, names in audit.duplicate_view_names.items())
        if audit.duplicate_view_names
        else "none"
    )
    duplicate_defaults = (
        "; ".join(f"{scope}={', '.join(names)}" for scope, names in audit.duplicate_default_views.items())
        if audit.duplicate_default_views
        else "none"
    )
    lines.append(f"- Duplicate view names: {duplicate_names}")
    lines.append(
        f"- Invalid view visibility: {', '.join(audit.invalid_visibility_views) if audit.invalid_visibility_views else 'none'}"
    )
    lines.append(
        f"- Views missing filters: {', '.join(audit.views_missing_filters) if audit.views_missing_filters else 'none'}"
    )
    lines.append(f"- Duplicate default views: {duplicate_defaults}")
    lines.append(
        f"- Orphan subscriptions: {', '.join(audit.orphan_subscriptions) if audit.orphan_subscriptions else 'none'}"
    )
    lines.append(
        "- Subscriptions missing recipients: "
        f"{', '.join(audit.subscriptions_missing_recipients) if audit.subscriptions_missing_recipients else 'none'}"
    )
    lines.append(
        "- Subscriptions with invalid channel: "
        f"{', '.join(audit.subscriptions_with_invalid_channel) if audit.subscriptions_with_invalid_channel else 'none'}"
    )
    lines.append(
        "- Subscriptions with invalid cadence: "
        f"{', '.join(audit.subscriptions_with_invalid_cadence) if audit.subscriptions_with_invalid_cadence else 'none'}"
    )
    return "\n".join(lines) + "\n"
