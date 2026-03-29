from dataclasses import dataclass, field
from datetime import datetime, timezone
from html import escape
from typing import Any, Dict, List, Optional, Sequence


def collaboration_now() -> str:
    return datetime.now(timezone.utc).isoformat().replace("+00:00", "Z")


@dataclass
class CollaborationComment:
    comment_id: str
    author: str
    body: str
    created_at: str = field(default_factory=collaboration_now)
    mentions: List[str] = field(default_factory=list)
    anchor: str = ""
    status: str = "open"

    def to_dict(self) -> Dict[str, Any]:
        return {
            "comment_id": self.comment_id,
            "author": self.author,
            "body": self.body,
            "created_at": self.created_at,
            "mentions": self.mentions,
            "anchor": self.anchor,
            "status": self.status,
        }

    @classmethod
    def from_dict(cls, data: Dict[str, Any]) -> "CollaborationComment":
        return cls(
            comment_id=str(data.get("comment_id", "")),
            author=str(data.get("author", "")),
            body=str(data.get("body", "")),
            created_at=str(data.get("created_at", collaboration_now())),
            mentions=[str(value) for value in data.get("mentions", [])],
            anchor=str(data.get("anchor", "")),
            status=str(data.get("status", "open")),
        )


@dataclass
class RepoPost:
    post_id: str
    channel: str
    author: str
    body: str
    target_surface: str = "task"
    target_id: str = ""
    parent_post_id: str = ""
    created_at: str = field(default_factory=collaboration_now)
    metadata: Dict[str, Any] = field(default_factory=dict)

    def to_dict(self) -> Dict[str, Any]:
        return {
            "post_id": self.post_id,
            "channel": self.channel,
            "author": self.author,
            "body": self.body,
            "target_surface": self.target_surface,
            "target_id": self.target_id,
            "parent_post_id": self.parent_post_id,
            "created_at": self.created_at,
            "metadata": dict(self.metadata),
        }

    @classmethod
    def from_dict(cls, data: Dict[str, Any]) -> "RepoPost":
        return cls(
            post_id=str(data.get("post_id", "")),
            channel=str(data.get("channel", "")),
            author=str(data.get("author", "")),
            body=str(data.get("body", "")),
            target_surface=str(data.get("target_surface", "task")),
            target_id=str(data.get("target_id", "")),
            parent_post_id=str(data.get("parent_post_id", "")),
            created_at=str(data.get("created_at", collaboration_now())),
            metadata=dict(data.get("metadata", {})),
        )

    def to_collaboration_comment(self) -> "CollaborationComment":
        return CollaborationComment(
            comment_id=f"repo-{self.post_id}",
            author=self.author,
            body=self.body,
            created_at=self.created_at,
            anchor=f"{self.target_surface}:{self.target_id}",
            status="resolved" if self.metadata.get("resolved") else "open",
        )


@dataclass
class RepoDiscussionBoard:
    posts: List[RepoPost] = field(default_factory=list)

    def create_post(
        self,
        *,
        channel: str,
        author: str,
        body: str,
        target_surface: str,
        target_id: str,
        metadata: Optional[Dict[str, Any]] = None,
    ) -> RepoPost:
        post = RepoPost(
            post_id=f"post-{len(self.posts) + 1}",
            channel=channel,
            author=author,
            body=body,
            target_surface=target_surface,
            target_id=target_id,
            metadata=dict(metadata or {}),
        )
        self.posts.append(post)
        return post

    def reply(self, *, parent_post_id: str, author: str, body: str) -> RepoPost:
        parent = next((post for post in self.posts if post.post_id == parent_post_id), None)
        if not parent:
            raise ValueError(f"unknown parent post: {parent_post_id}")
        post = RepoPost(
            post_id=f"post-{len(self.posts) + 1}",
            channel=parent.channel,
            author=author,
            body=body,
            target_surface=parent.target_surface,
            target_id=parent.target_id,
            parent_post_id=parent_post_id,
        )
        self.posts.append(post)
        return post

    def list_posts(self, *, channel: str = "", target_surface: str = "", target_id: str = "") -> List[RepoPost]:
        result = self.posts
        if channel:
            result = [post for post in result if post.channel == channel]
        if target_surface:
            result = [post for post in result if post.target_surface == target_surface]
        if target_id:
            result = [post for post in result if post.target_id == target_id]
        return list(result)


@dataclass
class DecisionNote:
    decision_id: str
    author: str
    outcome: str
    summary: str
    recorded_at: str = field(default_factory=collaboration_now)
    mentions: List[str] = field(default_factory=list)
    related_comment_ids: List[str] = field(default_factory=list)
    follow_up: str = ""

    def to_dict(self) -> Dict[str, Any]:
        return {
            "decision_id": self.decision_id,
            "author": self.author,
            "outcome": self.outcome,
            "summary": self.summary,
            "recorded_at": self.recorded_at,
            "mentions": self.mentions,
            "related_comment_ids": self.related_comment_ids,
            "follow_up": self.follow_up,
        }

    @classmethod
    def from_dict(cls, data: Dict[str, Any]) -> "DecisionNote":
        return cls(
            decision_id=str(data.get("decision_id", "")),
            author=str(data.get("author", "")),
            outcome=str(data.get("outcome", "")),
            summary=str(data.get("summary", "")),
            recorded_at=str(data.get("recorded_at", collaboration_now())),
            mentions=[str(value) for value in data.get("mentions", [])],
            related_comment_ids=[str(value) for value in data.get("related_comment_ids", [])],
            follow_up=str(data.get("follow_up", "")),
        )


@dataclass
class CollaborationThread:
    surface: str
    target_id: str
    comments: List[CollaborationComment] = field(default_factory=list)
    decisions: List[DecisionNote] = field(default_factory=list)

    @property
    def participant_count(self) -> int:
        participants = {
            comment.author for comment in self.comments
        } | {
            decision.author for decision in self.decisions
        }
        return len({value for value in participants if value})

    @property
    def mention_count(self) -> int:
        return sum(len(comment.mentions) for comment in self.comments) + sum(
            len(decision.mentions) for decision in self.decisions
        )

    @property
    def open_comment_count(self) -> int:
        return sum(1 for comment in self.comments if comment.status != "resolved")

    @property
    def recommendation(self) -> str:
        if self.decisions:
            return "share-latest-decision"
        if self.open_comment_count:
            return "resolve-open-comments"
        if self.comments:
            return "monitor-collaboration"
        return "no-collaboration-recorded"

    def to_dict(self) -> Dict[str, Any]:
        return {
            "surface": self.surface,
            "target_id": self.target_id,
            "comments": [comment.to_dict() for comment in self.comments],
            "decisions": [decision.to_dict() for decision in self.decisions],
        }

    @classmethod
    def from_dict(cls, data: Dict[str, Any]) -> "CollaborationThread":
        return cls(
            surface=str(data.get("surface", "")),
            target_id=str(data.get("target_id", "")),
            comments=[CollaborationComment.from_dict(value) for value in data.get("comments", [])],
            decisions=[DecisionNote.from_dict(value) for value in data.get("decisions", [])],
        )


def build_collaboration_thread(
    surface: str,
    target_id: str,
    *,
    comments: Optional[Sequence[CollaborationComment]] = None,
    decisions: Optional[Sequence[DecisionNote]] = None,
) -> CollaborationThread:
    return CollaborationThread(
        surface=surface,
        target_id=target_id,
        comments=list(comments or []),
        decisions=list(decisions or []),
    )


def merge_collaboration_threads(
    *,
    target_id: str,
    native_thread: Optional[CollaborationThread],
    repo_thread: Optional[CollaborationThread],
) -> Optional[CollaborationThread]:
    if native_thread is None and repo_thread is None:
        return None

    merged_comments: List[CollaborationComment] = []
    merged_decisions: List[DecisionNote] = []

    for thread in [native_thread, repo_thread]:
        if thread is None:
            continue
        merged_comments.extend(thread.comments)
        merged_decisions.extend(thread.decisions)

    merged_comments.sort(key=lambda item: item.created_at)
    merged_decisions.sort(key=lambda item: item.recorded_at)

    return CollaborationThread(
        surface="merged",
        target_id=target_id,
        comments=merged_comments,
        decisions=merged_decisions,
    )


def build_collaboration_thread_from_audits(
    audits: Sequence[Dict[str, Any]],
    *,
    surface: str,
    target_id: str,
) -> Optional[CollaborationThread]:
    comments: List[CollaborationComment] = []
    decisions: List[DecisionNote] = []

    for audit in audits:
        details = audit.get("details", {})
        audit_surface = str(details.get("surface", "run"))
        if audit_surface != surface:
            continue

        action = audit.get("action")
        if action == "collaboration.comment":
            comments.append(
                CollaborationComment(
                    comment_id=str(details.get("comment_id", "")),
                    author=str(audit.get("actor", "")),
                    body=str(details.get("body", "")),
                    created_at=str(audit.get("timestamp", collaboration_now())),
                    mentions=[str(value) for value in details.get("mentions", [])],
                    anchor=str(details.get("anchor", "")),
                    status=str(details.get("status", "open")),
                )
            )
        if action == "collaboration.decision":
            decisions.append(
                DecisionNote(
                    decision_id=str(details.get("decision_id", "")),
                    author=str(audit.get("actor", "")),
                    outcome=str(audit.get("outcome", "")),
                    summary=str(details.get("summary", "")),
                    recorded_at=str(audit.get("timestamp", collaboration_now())),
                    mentions=[str(value) for value in details.get("mentions", [])],
                    related_comment_ids=[str(value) for value in details.get("related_comment_ids", [])],
                    follow_up=str(details.get("follow_up", "")),
                )
            )

    if not comments and not decisions:
        return None

    return CollaborationThread(surface=surface, target_id=target_id, comments=comments, decisions=decisions)


def render_collaboration_lines(thread: Optional[CollaborationThread]) -> List[str]:
    if thread is None:
        return []

    lines = [
        "## Collaboration",
        "",
        f"- Surface: {thread.surface}",
        f"- Target: {thread.target_id}",
        f"- Participants: {thread.participant_count}",
        f"- Comments: {len(thread.comments)}",
        f"- Open Comments: {thread.open_comment_count}",
        f"- Mentions: {thread.mention_count}",
        f"- Decision Notes: {len(thread.decisions)}",
        f"- Recommendation: {thread.recommendation}",
    ]

    lines.extend(["", "## Comments", ""])
    if thread.comments:
        for comment in thread.comments:
            mentions = ", ".join(comment.mentions) if comment.mentions else "none"
            lines.append(
                f"- {comment.comment_id}: author={comment.author} status={comment.status} "
                f"anchor={comment.anchor or 'none'} mentions={mentions} body={comment.body}"
            )
    else:
        lines.append("- None")

    lines.extend(["", "## Decision Notes", ""])
    if thread.decisions:
        for decision in thread.decisions:
            mentions = ", ".join(decision.mentions) if decision.mentions else "none"
            related = ", ".join(decision.related_comment_ids) if decision.related_comment_ids else "none"
            lines.append(
                f"- {decision.decision_id}: outcome={decision.outcome} author={decision.author} "
                f"mentions={mentions} related={related} summary={decision.summary} follow_up={decision.follow_up or 'none'}"
            )
    else:
        lines.append("- None")

    lines.append("")
    return lines


def render_collaboration_panel_html(title: str, description: str, thread: Optional[CollaborationThread]) -> str:
    if thread is None:
        return f"""
        <section class="surface">
          <h2>{escape(title)}</h2>
          <p>{escape(description)}</p>
          <div class="empty">No collaboration recorded for this surface.</div>
        </section>
        """

    comment_items = "".join(
        f"""
        <article class="resource-card">
          <span class="kicker">{escape(comment.comment_id)}</span>
          <strong>{escape(comment.author)}</strong>
          <p>{escape(comment.body)}</p>
          <span class="resource-meta">status={escape(comment.status)} | anchor={escape(comment.anchor or 'none')} | mentions={escape(', '.join(comment.mentions) if comment.mentions else 'none')}</span>
        </article>
        """
        for comment in thread.comments
    ) or '<div class="empty">No comments recorded.</div>'
    decision_items = "".join(
        f"""
        <article class="resource-card" data-tone="report">
          <span class="kicker">{escape(decision.decision_id)}</span>
          <strong>{escape(decision.outcome)}</strong>
          <p>{escape(decision.summary)}</p>
          <span class="resource-meta">author={escape(decision.author)} | mentions={escape(', '.join(decision.mentions) if decision.mentions else 'none')} | related={escape(', '.join(decision.related_comment_ids) if decision.related_comment_ids else 'none')} | follow_up={escape(decision.follow_up or 'none')}</span>
        </article>
        """
        for decision in thread.decisions
    ) or '<div class="empty">No decision notes recorded.</div>'

    return f"""
    <section class="surface">
      <h2>{escape(title)}</h2>
      <p>{escape(description)}</p>
      <p class="meta">participants={thread.participant_count} | comments={len(thread.comments)} | open={thread.open_comment_count} | mentions={thread.mention_count} | decisions={len(thread.decisions)}</p>
    </section>
    <section class="surface">
      <h3>Comments</h3>
      <div class="resource-grid">{comment_items}</div>
    </section>
    <section class="surface">
      <h3>Decision Notes</h3>
      <div class="resource-grid">{decision_items}</div>
    </section>
    """
