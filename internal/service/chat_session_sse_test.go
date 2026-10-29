//
//  Copyright 2026 The InfiniFlow Authors. All Rights Reserved.
//
//  Licensed under the Apache License, Version 2.0 (the "License");
//  you may not use this file except in compliance with the License.
//  You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
//  Unless required by applicable law or agreed to in writing, software
//  distributed under the License is distributed on an "AS IS" BASIS,
//  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//  See the License for the specific language governing permissions and
//  limitations under the License.
//

package service

import (
	"strings"
	"testing"

	"ragflow/internal/agentic_rag"
)

// The SSE payload is assembled field by field, so a value present in the
// intermediate map only reaches the client if it is copied in here.
func TestSSEMarshalChunkCarriesTodo(t *testing.T) {
	state := &agentic_rag.TodoState{
		Todos: []agentic_rag.TodoItem{
			{Content: "locate the document", Status: agentic_rag.TodoCompleted},
			{Content: "read it", Status: agentic_rag.TodoInProgress},
		},
		Counts: agentic_rag.TodoCounts{Pending: 0, InProgress: 1, Completed: 1},
	}

	out := sseMarshalChunk(map[string]interface{}{"answer": "", "todo": state}, "")
	if !strings.Contains(out, `"todo"`) ||
		!strings.Contains(out, `"in_progress": 1`) ||
		!strings.Contains(out, `"locate the document"`) {
		t.Errorf("todo missing from the SSE chunk: %s", out)
	}
}

func TestSSEMarshalChunkOmitsAbsentTodo(t *testing.T) {
	out := sseMarshalChunk(map[string]interface{}{"answer": "hi"}, "")
	if strings.Contains(out, `"todo"`) {
		t.Errorf("todo must stay absent when the run has none: %s", out)
	}
}
