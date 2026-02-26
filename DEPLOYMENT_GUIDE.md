# Deploying to GitHub

This guide helps you push OiviaKind to GitHub for the first time.

## 📝 Pre-Push Checklist

Before pushing to GitHub, verify:

- [x] ✅ Latest.log created
- [x] ✅ Logs folder created  
- [x] ✅ .env file created (NOT committed)
- [x] ✅ .env.example committed (template for others)
- [x] ✅ .gitignore updated (protects sensitive files)
- [x] ✅ Git repository initialized
- [x] ✅ CONTRIBUTING.md added
- [x] ✅ GitHub templates created

## 🚀 Push to GitHub

### Step 1: Create GitHub Repository

Go to https://github.com/new and create a new repository named `OiviaKind` (or your preferred name).

**DO NOT** initialize with README, .gitignore, or license (we already have them).

### Step 2: Configure Git User (if not already done)

```sh
git config --global user.name "Your Name"
git config --global user.email "your.email@example.com"
```

### Step 3: Create Initial Commit

```sh
cd /home/oiviadesu/OiviaKind

# Add all files (excluding those in .gitignore)
git add -A

# Create initial commit
git commit -m "feat: initial commit of MusicService Operator

- Kubernetes operator for music streaming services
- MariaDB master/replica management with auto-replication
- HPA support for app and database
- Storage management (resize/recreate)
- Comprehensive testing suite
- GitHub templates and contribution guidelines"

# Verify what's committed
git log --stat
```

### Step 4: Add Remote and Push

```sh
# Add your GitHub repository as remote
git remote add origin https://github.com/<your-username>/OiviaKind.git

# Or using SSH:
# git remote add origin git@github.com:<your-username>/OiviaKind.git

# Push to main branch
git branch -M main
git push -u origin main
```

## 🔐 Security Verification

Double-check sensitive files are NOT being pushed:

```sh
# This should show ONLY .env.example, not .env
git ls-files | grep env

# This should show nothing
git ls-files | grep "\.log$"
git ls-files | grep "^logs/"
```

Expected output:
```
.env.example  ✅ (This is OK - it's a template)
```

If you see `.env` or `latest.log`, **STOP** and check your `.gitignore`!

## 📦 What's Being Pushed?

Your repository will include:

### ✅ Source Code
- `api/` - CRD definitions
- `internal/` - Controller logic
- `cmd/` - Entry point
- `config/` - Kubernetes manifests

### ✅ Documentation
- `README.md` - Project overview
- `CONTRIBUTING.md` - Contribution guidelines
- `TESTING_GUIDE.md` - Testing instructions
- This file

### ✅ Configuration
- `.gitignore` - Protects sensitive files
- `.env.example` - Environment template
- `Makefile` - Build automation
- `Dockerfile` - Container image

### ✅ GitHub Templates
- Issue templates (bug reports, feature requests)
- Pull request template
- Contribution workflow

### ❌ NOT Being Pushed (Protected)
- `.env` - Your actual secrets
- `latest.log` - Log files
- `logs/` - Log directory contents
- `bin/` - Downloaded binaries

## 🎯 After Pushing

1. **Add Repository Description** on GitHub
2. **Add Topics/Tags**: `kubernetes`, `operator`, `golang`, `music-streaming`, `mariadb`
3. **Enable Discussions** (Settings → Features)
4. **Setup Branch Protection** for `main` (optional)
5. **Add Repository Badges** to README (CI/CD, Go Report Card, etc.)

## 📄 License

Don't forget to choose a LICENSE! Common choices:
- MIT License (permissive)
- Apache 2.0 (permissive with patent grant)
- GPL v3 (copyleft)

Add via GitHub interface or create LICENSE file manually.

## 🤝 Collaboration

Share your repository link and invite collaborators:
```
https://github.com/<your-username>/OiviaKind
```

## 🆘 Troubleshooting

### Problem: "Permission denied (publickey)"
**Solution**: Setup SSH key or use HTTPS with personal access token.

### Problem: ".env accidentally committed"
**Solution**:
```sh
# Remove from git but keep locally
git rm --cached .env
git commit -m "fix: remove sensitive .env file"

# If already pushed, consider rotating secrets!
```

### Problem: "Large files rejected"
**Solution**: Check if any large binaries in `bin/` are being tracked.

---

Happy coding! 🎵✨
