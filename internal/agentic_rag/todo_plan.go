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
	"errors"
	"fmt"
	"strings"
	"sync"
)

// Todo statuses. They are the whole lifecycle the model and a UI need; no id,
// priority or activeForm, because a write replaces the entire list.
const (
	TodoPending    = "pending"
	TodoInProgress = "in_progress"
	TodoCompleted  = "completed"
)

// TodoItem is one entry of the run's plan.
type TodoItem struct {
	// Content is what the task is: a short imperative line.
	Content string `json:"content"`
	// Status is one of TodoPending / TodoInProgress / TodoCompleted.
	Status string `json:"status"`
}

// TodoCounts is the per-status tally shipped beside the list, so a client does
// not have to recount on every render.
type TodoCounts struct {
	Pending    int `json:"pending"`
	InProgress int `json:"in_progress"`
	Completed  int `json:"completed"`
}

// TodoState is the whole todo list after a write, plus its counts. Every
// snapshot is complete: replay is last-write-wins.
type TodoState struct {
	Todos  []TodoItem `json:"todos"`
	Counts TodoCounts `json:"counts"`
}

// todoPlan is the per-run todo state. The tool runs inside the run's context,
// so the state lives there too: nothing outlives one Run, and two concurrent
// conversations cannot see each other's list.
//
// Writes are serialized by the mutex because eino executes the tool calls of
// one model turn concurrently. onWrite is captured under the same lock and
// invoked outside it, so a slow consumer cannot block another tool call.
type todoPlan struct {
	mu      sync.Mutex
	todos   []TodoItem
	onWrite func(TodoState)
}

type todoPlanKey struct{}

// withTodoPlan installs a fresh plan in ctx. Every Run does this once, so a
// turn always starts with an empty list.
func withTodoPlan(ctx context.Context, onWrite func(TodoState)) context.Context {
	return context.WithValue(ctx, todoPlanKey{}, &todoPlan{onWrite: onWrite})
}

// todoPlanFrom returns the run's plan, or nil outside a run (a direct tool call
// in a test, for instance).
func todoPlanFrom(ctx context.Context) *todoPlan {
	plan, _ := ctx.Value(todoPlanKey{}).(*todoPlan)
	return plan
}

// replace installs the whole list, returns its snapshot, and reports it to the
// caller's listener.
func (p *todoPlan) replace(todos []TodoItem) TodoState {
	state := TodoState{Todos: append([]TodoItem(nil), todos...)}
	for _, todo := range todos {
		switch todo.Status {
		case TodoPending:
			state.Counts.Pending++
		case TodoInProgress:
			state.Counts.InProgress++
		case TodoCompleted:
			state.Counts.Completed++
		}
	}
	p.mu.Lock()
	p.todos = state.Todos
	onWrite := p.onWrite
	p.mu.Unlock()
	if onWrite != nil {
		onWrite(state)
	}
	return state
}

// canonicalTodos validates and normalizes a model-supplied list: trimmed
// non-empty unique content, a known status, and at most one in_progress task.
// The checks live here rather than in the JSON schema because they are
// whole-list rules the schema cannot express.
func canonicalTodos(raw []todoWriteItem) ([]TodoItem, error) {
	todos := make([]TodoItem, 0, len(raw))
	seen := make(map[string]bool, len(raw))
	active := 0
	for _, item := range raw {
		content := strings.TrimSpace(item.Content)
		if content == "" {
			return nil, errors.New("`content` must be a non-empty string")
		}
		if seen[content] {
			return nil, fmt.Errorf("duplicate content %q", content)
		}
		seen[content] = true
		switch item.Status {
		case TodoPending, TodoInProgress, TodoCompleted:
		default:
			return nil, fmt.Errorf("invalid status %q (want pending, in_progress or completed)", item.Status)
		}
		if item.Status == TodoInProgress {
			active++
		}
		todos = append(todos, TodoItem{Content: content, Status: item.Status})
	}
	if active > 1 {
		return nil, fmt.Errorf("at most one task may be in_progress (got %d)", active)
	}
	return todos, nil
}

// formatTodoState is the tool result the model reads back: the counts first,
// then the list, so the model can confirm what it just wrote without
// re-deriving the tally itself.
func formatTodoState(state TodoState) string {
	if len(state.Todos) == 0 {
		return "Todo list cleared."
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Todo list updated: %d tasks (%d pending, %d in progress, %d completed).\n",
		len(state.Todos), state.Counts.Pending, state.Counts.InProgress, state.Counts.Completed)
	for i, todo := range state.Todos {
		fmt.Fprintf(&b, "%d. [%s] %s\n", i+1, todo.Status, todo.Content)
	}
	return b.String()
}
