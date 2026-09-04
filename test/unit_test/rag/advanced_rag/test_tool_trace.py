import asyncio
import json

from rag.advanced_rag.harness.tool_trace import new_trace


def test_trace_has_explicit_unknown_scores(tmp_path, monkeypatch):
    path = tmp_path / "trace.jsonl"
    monkeypatch.setenv("RAGFLOW_TOOL_TRACE_PATH", str(path))

    async def run():
        trace = new_trace("request-1", research_round=2, slot_id=7)
        first = await trace.record(
            action="search_chunks",
            action_input={"query": ["x"]},
            session_turn=3,
            tool_batch_index=0,
            tool_result={"status": "ok", "reason": None, "evidence_ids": ["e1"]},
        )
        second = await trace.record(
            action="retrieve",
            action_input={"query": ["y"]},
            session_turn=3,
            tool_batch_index=1,
        )
        return first, second

    first, second = asyncio.run(run())
    assert [first["event_seq"], second["event_seq"]] == [1, 2]
    assert first["session_turn"] == second["session_turn"] == 3
    assert [first["tool_batch_index"], second["tool_batch_index"]] == [0, 1]
    assert first["confidence"] is None
    assert first["confidence_source"] == "not_provided"
    assert first["candidates"] == []
    assert first["metadata_transport"] == "not_provided"
    assert first["candidate_score_observed"] is False
    assert first["confidence_observed"] is False
    assert json.loads(path.read_text().splitlines()[0])["tool_result"]["evidence_ids"] == ["e1"]


def test_invalid_confidence_is_not_clamped(tmp_path, monkeypatch):
    path = tmp_path / "trace.jsonl"
    monkeypatch.setenv("RAGFLOW_TOOL_TRACE_PATH", str(path))

    async def run():
        return await new_trace("request-2").record(
            action="retrieve",
            action_input={},
            session_turn=1,
            tool_batch_index=0,
            confidence=2.0,
        )

    event = asyncio.run(run())
    assert event["confidence"] is None
    assert event["confidence_source"] == "parse_failed"
    assert event["confidence_observed"] is False
