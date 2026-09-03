Ask the user a structured question and wait for their response. Use this
when you need clarification, confirmation, or a choice before proceeding.

## How it works

Always provide a `questions` array with at least one item. A single item
renders as a plain question; multiple items render as a tabbed form with
a confirmation screen at the end.

Every question MUST include:
- `type` - `yes_no`, `single_choice`, `multi_choice`, or `free_text`
- `question` - a short, direct question (one line)
- `description` - markdown context shown below the question with details,
  tradeoffs, or examples. Always required. Omitting it causes an error.

## Hard limits

These are enforced. Violations return an error and waste a round trip.

- Max 5 choices per question. If you have more, group or prioritize.
- Choices required for `single_choice` and `multi_choice`.
- Description required on every question.
- Choice descriptions must be under 200 chars each.
- Max 5 questions per batch. If you need more, split into multiple
  batches and tell the user there will be follow-up questions.

## Question types

- `yes_no` - confirmation only. Use only for true accept/reject prompts.
- `single_choice` - pick one from `choices`. Use this for named
  alternatives, including binary ones like "automatic or manual?".
- `multi_choice` - pick one or more from `choices`.
- `free_text` - open-ended text input. No choices needed.

Single and multi choice questions automatically include a free-text
fill-in option so the user can type a custom answer. Do not add an
"Other", "Something else", or "Custom" choice manually.

## Confirmation screen

When asking multiple questions, a confirmation tab is shown after all
questions are answered.

- `confirm_title`: a short question like "Ready to go?"
- `confirm_description`: summarize what will happen based on the expected
  answers

## When to use

- Confirm destructive or ambiguous actions
- The user's request has multiple valid interpretations
- Need the user to pick from options
- Gather multiple related answers at once

## When NOT to use

- Questions answerable by reading code or docs
- Information obtainable via other tools
- Asking permission for tool execution
