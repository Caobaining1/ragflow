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
	"encoding/json"
	"errors"
	"fmt"

	einotool "github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

const todoWriteToolName = "todo_write"

// todoWriteToolDescription is the model-facing contract. It states the three
// rules the tool enforces: whole-list replacement, one active task, and
// retrieval-only scope.
const todoWriteToolDescription = `Record and update a structured task list for the current work. Call it once at the START of any question that needs more than one retrieval step, BEFORE the first search, with the concrete steps you intend. Send the ENTIRE list every call - it REPLACES the previous list (there are no partial updates, no per-item edits). Keep AT MOST ONE todo in_progress at a time; while work remains, exactly one active task should be in_progress. Mark a todo completed the moment it is done (do not batch completions), and leave no in_progress item only once all work is complete. Track RETRIEVAL and RESEARCH steps only - summarising findings and writing the answer belong to the think tool, not here. Skip the list for trivial single-step or purely conversational tasks. Statuses: pending (not started), in_progress (being worked on now), completed (finished).`

// todoWriteArgs is the JSON the model sends into InvokableRun.
type todoWriteArgs struct {
	Todos []todoWriteItem `json:"todos"`
}

// todoWriteItem is one model-supplied entry, kept unvalidated until
// canonicalTodos has checked the whole list.
type todoWriteItem struct {
	Content string `json:"content"`
	Status  string `json:"status"`
}

// TodoWriteTool replaces the run's whole todo list. It keeps no state of its
// own: the list lives in the run context (see todo_plan.go), which is what lets
// a client render the plan while the agent is still working.
type TodoWriteTool struct{}

// NewTodoWriteTool returns a TodoWriteTool implementing eino's tool.InvokableTool.
func NewTodoWriteTool() *TodoWriteTool {
	return &TodoWriteTool{}
}

// Info returns the tool's metadata for the chat model.
func (t *TodoWriteTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: todoWriteToolName,
		Desc: todoWriteToolDescription,
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"todos": {
				Type:     schema.Array,
				Desc:     "The COMPLETE task list, replacing any previous list.",
				Required: true,
				ElemInfo: &schema.ParameterInfo{
					Type: schema.Object,
					SubParams: map[string]*schema.ParameterInfo{
						"content": {
							Type:     schema.String,
							Required: true,
							Desc:     "What the task is - a short imperative line.",
						},
						"status": {
							Type:     schema.String,
							Required: true,
							Desc:     "Step state: pending, in_progress or completed.",
							Enum:     []string{TodoPending, TodoInProgress, TodoCompleted},
						},
					},
				},
			},
		}),
	}, nil
}

// InvokableRun validates the list, replaces the run's plan, and echoes the
// resulting counts and list back to the model.
func (t *TodoWriteTool) InvokableRun(ctx context.Context, argumentsInJSON string, _ ...einotool.Option) (string, error) {
	return guardedToolRun(ctx, todoWriteToolName, t.invokableRun, argumentsInJSON)
}

func (t *TodoWriteTool) invokableRun(ctx context.Context, argumentsInJSON string) (string, error) {
	var args todoWriteArgs
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		return "", fmt.Errorf("todo_write: parse arguments: %w", err)
	}
	todos, err := canonicalTodos(args.Todos)
	if err != nil {
		return "", fmt.Errorf("todo_write: %w", err)
	}
	plan := todoPlanFrom(ctx)
	if plan == nil {
		// The list is per-run state; without a run there is nowhere to write it,
		// and silently accepting the call would make the client view diverge
		// from what the model believes it recorded.
		return "", errors.New("todo_write: no todo plan in this run")
	}
	return formatTodoState(plan.replace(todos)), nil
}
