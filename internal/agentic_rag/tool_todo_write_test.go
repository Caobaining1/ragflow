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

package agentic_rag

import (
	"context"
	"strings"
	"testing"
)

func TestTodoWriteTool_ReplacesWholeList(t *testing.T) {
	var snapshots []TodoState
	ctx := withTodoPlan(context.Background(), func(state TodoState) {
		snapshots = append(snapshots, state)
	})
	tool := NewTodoWriteTool()

	out, err := tool.InvokableRun(ctx, `{"todos":[
		{"content":"  grep the corpus  ","status":"completed"},
		{"content":"read the document","status":"in_progress"},
		{"content":"compare the figures","status":"pending"}]}`)
	if err != nil {
		t.Fatalf("InvokableRun: %v", err)
	}
	if !strings.Contains(out, "3 tasks (1 pending, 1 in progress, 1 completed)") {
		t.Errorf("counts missing from the tool result: %q", out)
	}
	if !strings.Contains(out, "1. [completed] grep the corpus") {
		t.Errorf("content must be trimmed and echoed: %q", out)
	}
	if len(snapshots) != 1 || len(snapshots[0].Todos) != 3 {
		t.Fatalf("listener got %d snapshots: %+v", len(snapshots), snapshots)
	}

	// The second write replaces the first: no partial update, no merge.
	out, err = tool.InvokableRun(ctx, `{"todos":[{"content":"only step","status":"in_progress"}]}`)
	if err != nil {
		t.Fatalf("second InvokableRun: %v", err)
	}
	if strings.Contains(out, "grep the corpus") || !strings.Contains(out, "1 tasks") {
		t.Errorf("second write did not replace the list: %q", out)
	}
	if len(snapshots) != 2 || len(snapshots[1].Todos) != 1 {
		t.Fatalf("listener did not see the replacement: %+v", snapshots)
	}
}

func TestTodoWriteTool_ClearsList(t *testing.T) {
	ctx := withTodoPlan(context.Background(), nil)
	out, err := NewTodoWriteTool().InvokableRun(ctx, `{"todos":[]}`)
	if err != nil {
		t.Fatalf("InvokableRun: %v", err)
	}
	if out != "Todo list cleared." {
		t.Errorf("empty list must clear, got %q", out)
	}
}

func TestTodoWriteTool_RejectsInvalidLists(t *testing.T) {
	cases := []struct {
		name string
		args string
		want string
	}{
		{
			name: "blank content",
			args: `{"todos":[{"content":"   ","status":"pending"}]}`,
			want: "non-empty string",
		},
		{
			name: "duplicate content",
			args: `{"todos":[{"content":"a","status":"pending"},{"content":"a","status":"completed"}]}`,
			want: "duplicate content",
		},
		{
			name: "unknown status",
			args: `{"todos":[{"content":"a","status":"doing"}]}`,
			want: "invalid status",
		},
		{
			name: "two active tasks",
			args: `{"todos":[{"content":"a","status":"in_progress"},{"content":"b","status":"in_progress"}]}`,
			want: "at most one task may be in_progress",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := withTodoPlan(context.Background(), nil)
			out, err := NewTodoWriteTool().InvokableRun(ctx, tc.args)
			if err != nil {
				t.Fatalf("validation failure must become a result, got error: %v", err)
			}
			if !strings.Contains(out, toolErrorMarker) || !strings.Contains(out, tc.want) {
				t.Errorf("expected a %q <tool_error>, got %.200q", tc.want, out)
			}
		})
	}
}

func TestTodoWriteTool_RequiresARunPlan(t *testing.T) {
	out, err := NewTodoWriteTool().InvokableRun(context.Background(), `{"todos":[{"content":"a","status":"pending"}]}`)
	if err != nil {
		t.Fatalf("missing plan must become a result, got error: %v", err)
	}
	if !strings.Contains(out, toolErrorMarker) || !strings.Contains(out, "no todo plan") {
		t.Errorf("expected a missing-plan <tool_error>, got %.200q", out)
	}
}
