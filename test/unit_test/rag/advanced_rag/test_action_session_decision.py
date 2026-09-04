"""Regression tests for per-call tool decision metadata binding.

Covers the multi-tool batch bug where the parser keyed metadata by tool NAME
and only bound the single ``selected`` call, dropping every secondary call in
the batch to ``parse_failed``.
"""

from rag.advanced_rag.harness.action_session import (
    _parse_tool_decision_metadata,
    _validate_decision_entry,
)


def _call(cid, name, decision=None):
    return {"id": cid, "name": name, "args": {}, "decision_raw": decision}


def _env(thought, confidence, selected, candidates):
    return {
        "thought": thought,
        "confidence": confidence,
        "selected": selected,
        "candidates": candidates,
    }


def test_valid_single_envelope_binds_selected_call():
    calls = [_call("c1", "search_chunks"), _call("c2", "retrieve")]
    content = '{"thought":"need semantic recall","confidence":0.8,"candidates":[{"name":"search_chunks","score":0.8},{"name":"retrieve","score":0.2}],"selected":"search_chunks"}'
    result = _parse_tool_decision_metadata(content, calls)
    c1 = result["c1"]
    assert c1["source"] == "llm_self_reported_tool_selection"
    assert c1["thought"] == "need semantic recall"
    assert c1["confidence"] == 0.8
    assert len(c1["candidates"]) == 2
    # the OTHER call in the batch is NOT fabricated — it stays unobserved
    assert result["c2"]["source"] == "parse_failed"
    assert result["c2"]["thought"] is None
    assert result["c2"]["confidence"] is None
    assert result["c2"]["candidates"] == []


def test_per_call_decision_args_bind_every_call_in_batch():
    calls = [
        _call("a", "search_chunks", _env("use semantic", 0.7, "search_chunks", [{"name": "search_chunks", "score": 0.7}, {"name": "retrieve", "score": 0.3}])),
        _call("b", "retrieve", _env("use exact", 1.0, "retrieve", [{"name": "retrieve", "score": 1.0}])),
    ]
    result = _parse_tool_decision_metadata("", calls)
    assert result["a"]["source"] == "llm_self_reported_tool_selection"
    assert result["b"]["source"] == "llm_self_reported_tool_selection"
    assert result["a"]["confidence"] == 0.7
    assert result["b"]["confidence"] == 1.0


def test_calls_array_binds_by_call_id():
    calls = [_call("c1", "search_chunks"), _call("c2", "retrieve")]
    content = (
        '{"calls":['
        '{"call_id":"c1","thought":"a","confidence":0.7,"selected":"search_chunks",'
        '"candidates":[{"name":"search_chunks","score":0.7},{"name":"retrieve","score":0.3}]},'
        '{"call_id":"c2","thought":"b","confidence":0.6,"selected":"retrieve",'
        '"candidates":[{"name":"retrieve","score":1.0}]}'
        "]}"
    )
    result = _parse_tool_decision_metadata(content, calls)
    assert result["c1"]["source"] == "llm_self_reported_tool_selection"
    assert result["c2"]["source"] == "llm_self_reported_tool_selection"
    assert result["c1"]["confidence"] == 0.7
    assert result["c2"]["confidence"] == 0.6


def test_duplicate_names_do_not_collide():
    calls = [
        _call("d1", "search_chunks", _env("p", 0.5, "search_chunks", [{"name": "search_chunks", "score": 0.5}, {"name": "retrieve", "score": 0.5}])),
        _call("d2", "search_chunks"),
    ]
    result = _parse_tool_decision_metadata("", calls)
    assert result["d1"]["source"] == "llm_self_reported_tool_selection"
    # the second same-named call is a distinct entry, not overwritten
    assert result["d2"]["source"] == "provider_unsupported"


def test_selected_not_in_batch_is_rejected():
    calls = [_call("c1", "search_chunks")]
    content = '{"thought":"x","confidence":0.9,"selected":"list_chunks","candidates":[{"name":"list_chunks","score":1.0}]}'
    result = _parse_tool_decision_metadata(content, calls)
    assert result["c1"]["source"] == "parse_failed"


def test_score_sum_not_one_is_rejected():
    calls = [_call("c1", "search_chunks")]
    content = '{"thought":"x","confidence":0.9,"selected":"search_chunks","candidates":[{"name":"search_chunks","score":0.6},{"name":"retrieve","score":0.6}]}'
    result = _parse_tool_decision_metadata(content, calls)
    assert result["c1"]["source"] == "parse_failed"


def test_unknown_tool_name_in_candidates_is_rejected():
    calls = [_call("c1", "search_chunks")]
    content = '{"thought":"x","confidence":0.9,"selected":"search_chunks","candidates":[{"name":"search_chunks","score":0.7},{"name":"not_a_tool","score":0.3}]}'
    result = _parse_tool_decision_metadata(content, calls)
    assert result["c1"]["source"] == "parse_failed"


def test_out_of_range_confidence_is_rejected():
    calls = [_call("c1", "search_chunks")]
    content = '{"thought":"x","confidence":1.5,"selected":"search_chunks","candidates":[{"name":"search_chunks","score":1.0}]}'
    result = _parse_tool_decision_metadata(content, calls)
    assert result["c1"]["source"] == "parse_failed"


def test_empty_content_marks_provider_unsupported():
    calls = [_call("c1", "retrieve")]
    result = _parse_tool_decision_metadata("", calls)
    assert result["c1"]["source"] == "provider_unsupported"
    assert result["c1"]["thought"] is None
    assert result["c1"]["confidence"] is None


def test_validate_decision_entry_rejects_boolean_confidence():
    assert (
        _validate_decision_entry(
            {"thought": "x", "confidence": True, "selected": "retrieve", "candidates": [{"name": "retrieve", "score": 1.0}]},
            {"retrieve"},
        )
        is None
    )
