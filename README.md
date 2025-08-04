# 🍒 Cherry Picker

An interactive terminal-based Git cherry-pick tool that makes selectively applying commits from a source branch to a target branch intuitive and efficient.

![Cherry Picker Screenshot](shot.png)

## ✨ Features

### 🌿 Interactive Branch Selection
- **Dynamic branch selection at startup** - Choose source and target branches interactively
- **Local branches only** - Clean list showing only local branches (no remote duplicates)
- **Powerful search functionality** - Press `/` or `f` to search through branches
- **Real-time filtering** - Results update as you type each character
- **Smart navigation** - Use `j/k` or arrow keys to navigate filtered results
- **Graceful cancellation** - Press `q` to exit cleanly without error messages

*Screenshot showing branch selection with search functionality - coming soon*

### 🎯 Smart Commit Detection
- **Identifies commits in the source branch** that are not yet in the target branch
- **Filters by author** to show only your contributions (commits you authored)
- **Detects merge commits** and already-applied commits with visual indicators
- **Shows detailed metadata** including date, author, files changed, insertions/deletions
- **Cherry-picks selected commits** from source branch to target branch

### 🖱️ Interactive Selection
- **Individual selection**: Use `Space` or `Enter` to toggle commit selection
- **Range selection**: Press `r` to select multiple consecutive commits
- **Bulk operations**: `a` to select all, `c` to clear all selections
- Visual indicators for selected (✓), merge (🔀), and already-applied (✗) commits

### 🔍 Advanced Search & Filtering
- **Fuzzy search**: Press `/` or `f` to search through commits
- Search across commit messages, SHA hashes, author names, and changed files
- Real-time filtering with live search results
- Navigate search results with arrow keys

### 👁️ Detailed Commit Preview
- Press `p` or `Tab` to enter preview mode
- View full commit diffs with syntax highlighting
- See detailed statistics (insertions/deletions by file)
- Examine commit metadata and file changes
- Truncated diff view for large commits

### 🔄 Runtime Branch Switching
- **Source branch switching**: Press `B` to change the comparison branch during operation
- **Target branch switching**: Press `b` to change the destination branch during operation
- Lists available local branches for switching
- Automatically reloads commits when branches change

### ⚔️ Comprehensive Conflict Resolution
- Built-in conflict detection during cherry-pick operations
- Interactive conflict resolution interface
- Multiple resolution strategies per file:
  - Use "ours" version
  - Use "theirs" version
  - Open merge tool
  - Manual editing
- Visual conflict status indicators

### 🔧 Multiple Execution Modes
- **Cherry-pick mode** (`e`/`x`): Standard cherry-pick selected commits
- **Interactive rebase mode** (`i`): Launch Git's interactive rebase
- Automatic conflict handling with user guidance

## 🚀 Installation

### Prerequisites
- Git (installed and configured)

### Option 1: Download Pre-built Binaries (Recommended)

#### Download Latest Release
1. Go to [Releases](https://github.com/zechtz/cherry-picker/releases)
2. Download the binary for your platform:
   - **Linux (x64)**: `cherry-picker-linux-amd64`
   - **Linux (ARM64)**: `cherry-picker-linux-arm64`
   - **macOS (Intel)**: `cherry-picker-darwin-amd64`
   - **macOS (Apple Silicon)**: `cherry-picker-darwin-arm64`
   - **Windows (x64)**: `cherry-picker-windows-amd64.exe`

#### Install the Binary

**Linux/macOS:**
```bash
# Download (replace with your platform)
curl -L -o cherry-picker https://github.com/zechtz/cherry-picker/releases/latest/download/cherry-picker-linux-amd64

# Make executable
chmod +x cherry-picker

# Move to PATH (optional)
sudo mv cherry-picker /usr/local/bin/

# Or use locally
./cherry-picker
```

**Windows:**
```powershell
# Download using PowerShell
Invoke-WebRequest -Uri "https://github.com/zechtz/cherry-picker/releases/latest/download/cherry-picker-windows-amd64.exe" -OutFile "cherry-picker.exe"

# Run directly
.\cherry-picker.exe

# Or add to PATH and run globally
cherry-picker
```

#### Verify Installation
```bash
# Check version
cherry-picker --version

# Or run in any git repository
cd /path/to/your/git/repo
cherry-picker
```

#### Quick Start
```bash
# 1. Navigate to your git repository
cd /path/to/your/project

# 2. Run cherry-picker
cherry-picker

# 3. Select source branch (where commits come from)
# 4. Select target branch (where commits will be applied) 
# 5. Choose commits to cherry-pick using Space/Enter
# 6. Press 'e' to execute the cherry-pick
```

#### Security: Verify Checksums (Optional but Recommended)
```bash
# Download checksums file
curl -L -o checksums.txt https://github.com/zechtz/cherry-picker/releases/latest/download/checksums.txt

# Verify your downloaded binary (Linux example)
sha256sum -c checksums.txt --ignore-missing
# Should show: cherry-picker-linux-amd64: OK
```

### Option 2: Build from Source
*Requires Go 1.23.0 or later*

```bash
git clone https://github.com/zechtz/cherry-picker.git
cd cherry-picker
go build -o cherry-picker

# Install globally (optional)
sudo mv cherry-picker /usr/local/bin/
```

### Option 3: Using Package Managers

#### Homebrew (macOS/Linux)
```bash
# Coming soon...
brew install zechtz/tap/cherry-picker
```

#### Chocolatey (Windows)
```bash
# Coming soon...
choco install cherry-picker
```

## 🎮 Usage

### Basic Usage
```bash
# Run in any git repository
cherry-picker

# Start with commits in reverse order
cherry-picker --reverse

# Generate default configuration file
cherry-picker --generate-config
```

### Workflow Example
1. Navigate to any branch (your current branch doesn't matter for commit selection)
2. Run `cherry-picker`
3. **Select source branch** - Choose branch containing commits you want to cherry-pick (e.g., `dev`, `main`)
4. **Select target branch** - Choose destination for cherry-picking (e.g., `staging`)
5. Tool shows commits from source branch that are NOT in target branch
6. Select commits using `Space` or `Enter`
7. Use `/` to search for specific commits if needed
8. Press `e` to execute cherry-pick (applies selected commits from source to target)
9. Handle any conflicts in the resolution interface

## ⌨️ Keyboard Shortcuts

### Branch Selection (Startup)
| Key | Action |
|-----|--------|
| `↑/↓` or `j/k` | Navigate through branches |
| `Enter/Space` | Select highlighted branch |
| `/` or `f` | Enter search mode |
| `q/Ctrl+C` | Quit branch selection |

### Branch Search Mode
| Key | Action |
|-----|--------|
| `Type` | Filter branches in real-time |
| `Enter` | Exit search mode (keep filter) |
| `Esc` | Clear search and exit search mode |
| `Backspace` | Remove last character from search |

### Navigation (Main UI)
| Key | Action |
|-----|--------|
| `↑/↓` or `j/k` | Move cursor up/down |
| `Page Up/Down` | Jump by page |
| `Home/End` | Go to first/last commit |

### Selection
| Key | Action |
|-----|--------|
| `Space/Enter` | Toggle commit selection |
| `r` | Toggle range selection mode |
| `a` | Select all commits |
| `c` | Clear all selections |

### Views & Modes
| Key | Action |
|-----|--------|
| `d` | Toggle detail view |
| `p/Tab` | Toggle preview mode |
| `/` or `f` | Enter search mode |
| `R` | Reverse commit order |

### Branch Management
| Key | Action |
|-----|--------|
| `b` | Switch target branch |
| `B` | Switch source branch |

### Execution
| Key | Action |
|-----|--------|
| `e/x` | Execute cherry-pick |
| `i` | Interactive rebase mode |
| `q/Ctrl+C` | Quit |

### Commit Search Mode
| Key | Action |
|-----|--------|
| `Type` | Filter commits |
| `Enter` | Exit search mode |
| `Esc` | Clear search and exit |

## ⚙️ Configuration

Cherry Picker uses a YAML configuration file located at `~/.cherry-picker.yaml`.

### Generate Default Config
```bash
cherry-picker --generate-config
```

### Configuration Options

```yaml
git:
  # Default target branch for cherry-picking
  target_branch: "clean-staging"
  
  # Default source branch to compare against
  source_branch: "dev"
  
  # Remote name
  remote: "origin"
  
  # Automatically fetch remote before operations
  auto_fetch: true
  
  # Branches where the tool should not run
  excluded_branches:
    - "main"
    - "master"
    - "production"

ui:
  # Cursor blink interval in milliseconds
  cursor_blink_interval: 500
  
  # Show commit date in the list
  show_commit_date: false
  
  # Show commit author in the list
  show_commit_author: false
  
  # Maximum number of commits to display
  max_commits: 100

behavior:
  # Start in reverse order by default
  default_reverse: false
  
  # Require confirmation before executing
  require_confirmation: true
  
  # Automatically push after successful cherry-pick
  auto_push: false
```

## 🛠️ Development

### Project Structure
```
├── main.go          # Entry point and CLI flag handling
├── models.go        # Data structures and core logic
├── tui.go          # Bubbletea UI implementation
├── git.go          # Git operations and command execution
├── config.go       # Configuration management
├── app.go          # Application setup and initialization
└── smart-cherry-pick.sh  # Legacy shell script reference
```

### Dependencies
- [Bubbletea](https://github.com/charmbracelet/bubbletea) - Terminal UI framework
- [Lipgloss](https://github.com/charmbracelet/lipgloss) - Style and layout
- [YAML v3](https://gopkg.in/yaml.v3) - Configuration parsing

### Building
```bash
go mod tidy
go build -o cherry-picker
```

### Testing
```bash
go test ./...
```

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## 📝 License

This project is licensed under the MIT License - see the LICENSE file for details.

## 🙏 Acknowledgments

- Built with [Charm](https://charm.sh/) TUI libraries
- Inspired by the need for better Git cherry-pick workflows
- Thanks to the Go community for excellent tooling

---

**Happy cherry-picking! 🍒**