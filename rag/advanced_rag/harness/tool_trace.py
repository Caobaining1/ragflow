"""Structured, request-local tool-call trace writer.

Native provider tool calls do not expose calibrated tool probabilities.  This
module therefore records only observed decision metadata and uses explicit
null/source fields when it is unavailable.
"""
from __future__ import annotations

import asyncio
import json
import logging
import os
import time
from dataclasses import dataclass, field

_LOG = logging.getLogger(__name__)


@dataclass
class ToolTrace:
    trace_id: str
    research_round: int | None = None
    slot_id: str | int | None = None
    _event_seq: int = 0
    _lock: asyncio.Lock = field(default_factory=asyncio.Lock, repr=False)

    async def record(
        self,
        *,
        action: str,
        action_input: dict,
        session_turn: int,
        tool_batch_index: int,
        slot_id=None,
        research_round=None,
        phase: str = "action_session",
        tool_result: dict | None = None,
        thought: str | None = None,
        candidates: list | None = None,
        metadata_source: str = "not_provided",
        confidence: float | None = None,
        confidence_source: str = "not_provided",
    ) -> dict:
        if confidence is not None and not 0 <= confidence <= 1:
            confidence = None
            confidence_source = "parse_failed"
        observed = metadata_source == "llm_self_reported_tool_selection" and confidence is not None and bool(thought) and bool(candidates)
        async with self._lock:
            self._event_seq += 1
            event = {
                "trace_id": self.trace_id,
                "event_seq": self._event_seq,
                "ts": time.strftime("%Y-%m-%dT%H:%M:%S%z"),
                "research_round": self.research_round if research_round is None else research_round,
                "slot_id": self.slot_id if slot_id is None else slot_id,
                "session_turn": session_turn,
                "step": session_turn,
                "step_semantics": "session_turn",
                "tool_batch_index": tool_batch_index,
                "phase": phase,
                "action": action,
                "action_input": action_input,
                "thought": thought,
                "selected": True,
                "confidence": confidence,
                "confidence_source": confidence_source,
                "confidence_semantics": "llm_self_reported_tool_selection",
                "confidence_observed": observed,
                "candidates": candidates or [],
                "candidate_score_observed": observed,
                "score_semantics": "llm_self_reported_candidate_score",
                "metadata_parse_source": metadata_source,
                "tool_result": tool_result or {"status": None, "reason": None, "evidence_ids": []},
            }
            path = os.environ.get("RAGFLOW_TOOL_TRACE_PATH", "/ragflow/logs/tool_trace.jsonl")
            try:
                with open(path, "a", encoding="utf-8") as stream:
                    stream.write(json.dumps(event, ensure_ascii=False, default=str) + "\n")
            except Exception:  # noqa: BLE001
                _LOG.warning("[ToolTrace] failed to write tool decision trace", exc_info=True)
            return event


def new_trace(trace_id: str, *, research_round: int | None = None, slot_id=None) -> ToolTrace:
    return ToolTrace(trace_id=str(trace_id), research_round=research_round, slot_id=slot_id)
