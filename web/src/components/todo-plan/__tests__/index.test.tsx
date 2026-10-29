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

import { render, screen } from '@testing-library/react';
import TodoPlan from '../index';

// Untyped on purpose: files with jest.mock are transformed by esbuild-jest's
// babel pass, which cannot strip type-only import usages.
const threeTasks = {
  todos: [
    { content: 'grep the corpus', status: 'completed' },
    { content: 'read the document', status: 'in_progress' },
    { content: 'compare the figures', status: 'pending' },
  ],
  counts: { pending: 1, in_progress: 1, completed: 1 },
};

const twoTasks = {
  todos: [
    { content: 'done task', status: 'completed' },
    { content: 'active task', status: 'in_progress' },
  ],
  counts: { pending: 0, in_progress: 1, completed: 1 },
};

// The component only needs `t` to return something stable: this file asserts
// the plan itself, not the translated labels.
jest.mock('react-i18next', () => ({
  useTranslation: () => ({
    t: (key: string, vars?: { done: number; total: number }) =>
      vars ? `${key}:${vars.done}/${vars.total}` : key,
  }),
}));

describe('TodoPlan', () => {
  it('renders every task and the completion count', () => {
    render(<TodoPlan todo={threeTasks as any} />);

    expect(screen.getByText('grep the corpus')).toBeInTheDocument();
    expect(screen.getByText('read the document')).toBeInTheDocument();
    expect(screen.getByText('compare the figures')).toBeInTheDocument();
    expect(screen.getByText('chat.todoProgress:1/3')).toBeInTheDocument();
  });

  it('dims completed tasks without striking them through', () => {
    render(<TodoPlan todo={twoTasks as any} />);

    expect(screen.getByText('done task').className).not.toContain(
      'line-through',
    );
    expect(screen.getByText('done task').className).toContain(
      'text-text-secondary',
    );
  });

  it('renders nothing for an empty list', () => {
    const { container } = render(
      <TodoPlan
        todo={
          {
            todos: [],
            counts: { pending: 0, in_progress: 0, completed: 0 },
          } as any
        }
      />,
    );

    expect(container).toBeEmptyDOMElement();
  });
});
