/*
 *  Copyright 2026 The InfiniFlow Authors. All Rights Reserved.
 *
 *  Licensed under the Apache License, Version 2.0 (the "License");
 *  you may not use this file except in compliance with the License.
 *  You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 *  Unless required by applicable law or agreed to in writing, software
 *  distributed under the License is distributed on an "AS IS" BASIS,
 *  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *  See the License for the specific language governing permissions and
 *  limitations under the License.
 */

import { ITodoItem, ITodoState } from '@/interfaces/database/chat';
import { cn } from '@/lib/utils';
import { Check, Circle, Loader2 } from 'lucide-react';
import { memo } from 'react';
import { useTranslation } from 'react-i18next';

function TodoStatusIcon({ status }: { status: ITodoItem['status'] }) {
  if (status === 'completed') {
    return <Check className="mt-0.5 size-3.5 shrink-0 text-text-secondary" />;
  }
  if (status === 'in_progress') {
    return (
      <Loader2 className="mt-0.5 size-3.5 shrink-0 animate-spin text-text-primary" />
    );
  }
  return <Circle className="mt-0.5 size-3.5 shrink-0 text-text-disabled" />;
}

/**
 * The agentic run's plan, rendered from the whole-list snapshots the
 * todo_write tool emits. Each snapshot replaces the previous one, so this is a
 * pure function of the latest state — there is nothing to accumulate here.
 */
const TodoPlan = ({ todo }: { todo: ITodoState }) => {
  const { t } = useTranslation();

  if (todo.todos.length === 0) {
    return null;
  }

  return (
    <section
      className="mb-2 rounded-md border border-border-button bg-bg-card px-3 py-2"
      aria-label={t('chat.todoTitle')}
    >
      <header className="flex items-center justify-between gap-2 text-xs text-text-secondary">
        <span>{t('chat.todoTitle')}</span>
        <span>
          {t('chat.todoProgress', {
            done: todo.counts.completed,
            total: todo.todos.length,
          })}
        </span>
      </header>
      <ul className="mt-1.5 flex flex-col gap-1">
        {todo.todos.map((item) => (
          <li key={item.content} className="flex items-start gap-2 text-sm">
            <TodoStatusIcon status={item.status} />
            <span
              className={cn(
                'break-words text-text-primary',
                // A finished task is dimmed, never struck through: the line
                // makes a short plan unreadable and hides the wording the
                // user wants to re-check.
                item.status === 'completed' && 'text-text-secondary',
              )}
            >
              {item.content}
            </span>
          </li>
        ))}
      </ul>
    </section>
  );
};

export default memo(TodoPlan);
