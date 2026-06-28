# c0d3r Tool Reference

Complete guide to all tools available inside c0d3r interactive sessions.

## Overview

Tools are special commands you can use while in an interactive c0d3r session. They let you:
- View and edit files
- Run shell commands
- Search for files
- Create/delete files
- Explore directory structure

Tools start with `/` and can be used anywhere in your prompts.

---

## File Tools

### `/view` — View file contents

Display the contents of a file.

**Syntax:**
```
/view <file>
/view <file>:<start_line>:<end_line>
/view <file>:<around_line>
```

**Examples:**

View entire file:
```
/view src/main.py
```

View specific lines:
```
/view src/main.py:10:20     # Lines 10-20
/view src/main.py:5:15      # Lines 5-15
```

View around a line:
```
/view src/main.py:100       # Lines around 100 (±10 lines)
```

**Output:**
```
File: src/main.py (215 lines)

1   import sys
2   from pathlib import Path
3   
4   def main():
5       print("Hello, world!")
...
```

---

### `/edit` — Edit file

Modify file contents.

**Syntax:**
```
/edit <file>
<old_text>
---
<new_text>
```

**Example 1: Simple replacement**
```
/edit src/main.py
print("hello")
---
print("Hello, World!")
```

**Example 2: Add function**
```
/edit src/utils.py
def existing_function():
    pass
---
def existing_function():
    pass

def new_helper_function():
    """Helper function."""
    return True
```

**Example 3: Add imports**
```
/edit src/main.py
import sys
---
import sys
import json
from pathlib import Path
```

**Tips:**
- Indent code correctly
- The old text must be exact (whitespace matters)
- Can be used multiple times in one session
- Claude can suggest edits for you

---

### `/create` — Create new file

Create a new file with initial content.

**Syntax:**
```
/create <path>
<content>
```

**Example:**
```
/create src/config.py
"""Configuration module."""

DEBUG = True
MAX_RETRIES = 3
TIMEOUT = 30
```

**Example 2: With code structure**
```
/create tests/test_main.py
import unittest
from src.main import main

class TestMain(unittest.TestCase):
    def test_main(self):
        result = main()
        self.assertIsNotNone(result)

if __name__ == "__main__":
    unittest.main()
```

---

### `/delete` — Delete file or directory

Remove a file or directory.

**Syntax:**
```
/delete <path>
/delete -r <path>         # Recursive (for directories)
```

**Examples:**

Delete single file:
```
/delete src/old_module.py
```

Delete directory:
```
/delete -r src/temp/
```

**Warning:** This is permanent. Use with caution.

---

## Shell Tools

### `/bash` — Execute shell command

Run any shell command and capture output.

**Syntax:**
```
/bash <command>
```

**Examples:**

List files:
```
/bash ls -la src/
```

Run build:
```
/bash npm run build
```

Check git status:
```
/bash git status
```

Run tests:
```
/bash python -m pytest tests/ -v
```

Install package:
```
/bash pip install requests
```

Run Python script:
```
/bash python src/main.py --help
```

Make HTTP request:
```
/bash curl https://api.github.com/repos/owner/repo
```

Chain commands:
```
/bash git add . && git commit -m "Update" && git push
```

**Output:**
The command's stdout and stderr are captured and displayed.

---

## Search Tools

### `/find` — Find files

Search for files matching patterns.

**Syntax:**
```
/find <pattern>
/find -name <name>
/find -type <type>
/find -path <path>
```

**Examples:**

Find all Python files:
```
/find *.py
```

Find test files:
```
/find -name test_*.py
```

Find directories:
```
/find -type d src/
```

Find in specific path:
```
/find -path src/*.py
```

Find by extension:
```
/find -name "*.ts"
/find -name "*.tsx"
/find -name "*.json"
```

**Output:**
```
src/main.py
src/utils.py
tests/test_main.py
```

---

### `/grep` — Search file contents

Search for text patterns in files.

**Syntax:**
```
/grep <pattern> <path>
/grep -r <pattern> <path>    # Recursive
/grep -i <pattern> <path>    # Case-insensitive
```

**Examples:**

Find TODO comments:
```
/grep "TODO" src/
```

Recursive search:
```
/grep -r "function_name" .
```

Case-insensitive:
```
/grep -i "error" src/ -r
```

Find imports:
```
/grep "import" *.py
```

Find specific pattern:
```
/grep "class\|def" src/ -r
```

**Output:**
```
src/main.py:42:    # TODO: refactor this
src/utils.py:15:   # TODO: add error handling
tests/test_main.py:108: def test_xyz():
```

---

## Directory Tools

### `/tree` — Show directory structure

Display directory contents as a tree.

**Syntax:**
```
/tree <path>
/tree <path> --depth <n>
```

**Examples:**

Show current structure:
```
/tree .
```

Show specific directory:
```
/tree src/
```

Limit depth:
```
/tree . --depth 2
```

**Output:**
```
src/
├── main.py
├── utils.py
├── config.py
└── commands/
    ├── login.py
    ├── logout.py
    └── help.py
```

---

## Advanced Usage

### Combining Tools in Prompts

You can use multiple tools in a single prompt:

```
/view src/main.py
/find -name test_*.py
/bash python -m pytest tests/ -v

Now please:
1. Review the test results
2. Update main.py to fix any failures
3. Run tests again to verify
```

### Using Tool Output in Prompts

Claude can see and reference tool outputs:

```
/bash git log --oneline -5
/view src/main.py:1:20

Based on the recent commits and current code, 
please add proper error handling to this function.
```

### Iterative Development

Common pattern for iterative improvement:

```
1. /view src/module.py           # See current code
2. "Please refactor this"         # Ask Claude to improve
3. Claude suggests changes        # Review suggestions
4. /edit src/module.py           # Apply changes
5. /bash python -m pytest        # Test
6. Review results, iterate
```

### Debugging Workflow

```
1. /bash python src/main.py      # Run and see error
2. /view src/main.py:100:120     # Look at relevant code
3. "Why is this failing?"         # Ask Claude
4. /edit src/main.py             # Apply fix
5. /bash python src/main.py      # Test again
```

---

## Tips & Best Practices

### Use Relative Paths

Tools work best with relative paths from your working directory:

```
/view src/main.py           # ✅ Good
/view /absolute/path/main.py # ⚠️  Works but less portable
```

### Line Numbers are Helpful

When you know approximately where to look:

```
/view src/main.py:100:120   # Faster than viewing entire file
```

### Filter Before Viewing

For large files, use `/grep` or `/bash` to find what you need:

```
/bash grep -n "def function_name" src/main.py  # Find line number
/view src/main.py:42:52                        # Then view that area
```

### Use `/bash` for Complex Operations

For operations harder to express with basic tools:

```
/bash find . -name "*.py" -exec wc -l {} + | sort -n
/bash git diff HEAD~1..HEAD -- src/
/bash npm outdated
```

### Check Before Deleting

Always verify before using `/delete`:

```
/find -name "*.tmp"     # First, find what will be deleted
/delete -r src/tmp/     # Then delete
```

---

## Keyboard Shortcuts in Tool Output

When viewing file contents or tool output:

- `Space` / `Page Down` — Scroll down
- `b` / `Page Up` — Scroll up
- `g` — Go to top
- `G` — Go to bottom
- `/<pattern>` — Search within output
- `q` — Quit (if in pager)

---

## Error Messages

### File Not Found
```
Error: file not found: src/nonexistent.py
```
**Solution:** Check the path and try `/find` to locate the file.

### Permission Denied
```
Error: permission denied: /etc/passwd
```
**Solution:** Use `sudo` with `/bash` if needed, or check file permissions.

### Pattern Not Found
```
No matches found for pattern: "nonexistent_function"
```
**Solution:** The pattern doesn't exist. Try different search terms or use `/find`.

### Directory Not Empty
```
Error: directory not empty: src/temp/
```
**Solution:** Use `/delete -r` to recursively delete, or manually delete contents first.

---

## Performance Notes

### Large Files

For very large files (>10,000 lines):
```
/bash wc -l large_file.py           # Check size first
/bash grep -n "target" large_file.py # Find line number
/view large_file.py:100:110         # View specific section
```

### Deep Directories

For deep directory trees:
```
/tree . --depth 1        # Limit depth
/find -type f -name "*.py"  # Or use find to search
```

### Many Results

If `/find` or `/grep` returns many results:
```
/grep "pattern" src/ | head -20      # Limit output
/find -name "*.py" | wc -l           # Count matches
```

---

## Integration with Claude

Claude can:
- See all tool outputs
- Suggest tool usage
- Create new files with `/create`
- Modify files with `/edit`
- Run tests with `/bash`
- Analyze search results

Example conversation:

```
You: /view src/main.py

Claude: I can see the code. Let me search for related functions.
Claude: /bash grep -r "call_main" src/

Claude: Based on the code structure, I recommend:
1. Refactoring duplicate logic
2. Adding error handling
3. Adding type hints

Should I apply these changes?

You: Yes, please. Then run tests.

Claude: 
/edit src/main.py
[... makes changes ...]

/bash python -m pytest tests/ -v
[... shows test results ...]
```

---

## See Also

- [COMMANDS.md](COMMANDS.md) — All CLI commands
- [API.md](API.md) — Detailed API documentation
- [README.md](README.md) — Project overview
