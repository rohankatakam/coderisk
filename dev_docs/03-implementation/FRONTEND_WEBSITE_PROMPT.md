# Claude Code Prompt: Frontend Website Updates

**Session Type:** Frontend (coderisk-frontend repository)
**Estimated Time:** 2-3 hours
**Phase:** Pre-Launch Website Update
**Reference Docs:** [website_messaging.md](website_messaging.md), [open_core_strategy.md](../00-product/open_core_strategy.md)

---

## Context

You are updating the CodeRisk website (coderisk.dev) to reflect the new **open core positioning**. The website currently shows CodeRisk as a general "trust infrastructure" but needs to emphasize:

1. **Open Source** - CLI and local mode are MIT licensed
2. **Easy to Install** - Homebrew one-command setup
3. **Optional Cloud** - Paid platform for teams

Read these documents first for complete context:
- [website_messaging.md](website_messaging.md) - Complete website content strategy
- [open_core_strategy.md](../00-product/open_core_strategy.md) - Open source positioning
- [packaging_and_distribution.md](packaging_and_distribution.md) - Installation methods

---

## Objective

Update the homepage to communicate the open source positioning, add new pages for pricing and open source details, and ensure all installation methods are documented.

---

## Tasks

### 1. Homepage Updates (`app/page.tsx`)

**Update Hero Section:**

Change headline from:
```
Trust infrastructure for AI-generated code
```

To:
```
Open Source AI Code Risk Assessment
```

Add subheadline:
```
The pre-flight check for developers. Know if your code is safe before you commit.
Free CLI, optional cloud platform.
```

Add badges below headline:
- MIT Licensed (link to GitHub LICENSE)
- GitHub stars (dynamic, fetch from GitHub API if possible)
- "Updated daily" or similar trust signal

**Update CTAs:**
- Primary: "Install CLI" (scroll to install section)
- Secondary: "View Docs" (link to /docs or external docs)

**Update Quick Start:**

Change from:
```bash
npm install -g coderisk
```

To:
```bash
brew tap rohankatakam/coderisk
brew install crisk
```

Show platform tabs if possible (macOS, Linux, Windows) with different commands.

**Add Open Source Section (new):**

After "How It Works" section, add:

```
# Open Source

CodeRisk CLI is **free and open source** (MIT License).

**What's included:**
- ✅ CLI tool (full source code)
- ✅ Local mode (run on your machine)
- ✅ Phase 1 metrics (coupling, co-change, test coverage)
- ✅ Pre-commit hooks
- ✅ Docker Compose stack

[View on GitHub] [Read License]
```

**Add Cloud Platform Teaser (new):**

After Open Source section:

```
# Need More? Try Cloud

For teams wanting zero setup and advanced features:

- ⚡ Instant access (no Docker, no database setup)
- 👥 Team collaboration (shared graphs)
- 🎯 ARC Database (100+ risk patterns from 10K incidents)
- 🤖 Phase 2 LLM (advanced investigation)

**Pricing:** $10-50/user/month

[Compare Plans] [Start Free Trial]
```

**Update Installation Instructions:**

Add comprehensive section with 4 methods:

1. **Homebrew** (macOS/Linux) - recommended
2. **Install Script** (curl one-liner)
3. **Docker** (containerized)
4. **Direct Download** (links to GitHub Releases)

See website_messaging.md Section "Detailed Page Content" for exact copy.

**Update Feature Grid:**

Add 4th card:
```
🔓 Open Source
MIT Licensed
Audit the code, contribute features, run locally forever
```

Ensure existing cards match messaging:
- Fast: 2-5 seconds
- Accurate: <3% false positives
- Easy: brew install crisk

### 2. Navigation Updates

**Add links to navigation bar:**
- "Open Source" (link to `/open-source`)
- "Pricing" (link to `/pricing`)
- Keep: "Docs", "GitHub"

Update GitHub link to: `https://github.com/rohankatakam/coderisk-go`

### 3. Create Pricing Page (`app/pricing/page.tsx`)

Create new page with structure from website_messaging.md Section "Pricing Page".

**Tiers:**
1. **Free Forever** - $0/month (CLI, local mode)
2. **Starter** - $10/user/month (cloud, private repos)
3. **Pro** - $25/user/month (team features)
4. **Enterprise** - $50/user/month (self-hosted, SSO)

Include FAQs:
- Is the CLI really free?
- What's the difference between local and cloud?
- Can I upgrade/downgrade anytime?
- Do you offer discounts?

See website_messaging.md for complete copy.

### 4. Create Open Source Page (`app/open-source/page.tsx`)

Create new page explaining the open core model.

Sections:
1. **Philosophy** - Why open core
2. **What's Open Source** - List of MIT licensed components
3. **What's Proprietary** - Cloud platform features
4. **Contributing** - Link to CONTRIBUTING.md
5. **License** - Link to LICENSE
6. **Community** - Links to GitHub, Discord, Twitter

See website_messaging.md Section "Open Source Page" for complete copy.

### 5. Add Install Script to Public Assets

**Copy `install.sh` to `public/install.sh`:**

This script will be created in the backend session. Once available, copy it to the frontend repo so it's accessible at:
```
https://coderisk.dev/install.sh
```

**For now, create placeholder:**
```bash
#!/bin/bash
# CodeRisk CLI Installer
# TODO: Replace with actual install script from backend repo
echo "Install script coming soon!"
echo "For now, use: brew tap rohankatakam/coderisk && brew install crisk"
```

### 6. Footer Updates

**Add Open Source section to footer:**

```
[Open Source]
- GitHub
- Contributing
- License
- Roadmap
```

**Add legal disclaimer:**

Update footer copyright to include:
```
© 2025 CodeRisk. Open Source (MIT License).
```

### 7. Meta Tags & SEO

**Update `app/layout.tsx` metadata:**

```typescript
export const metadata = {
  title: 'CodeRisk - Open Source AI Code Risk Assessment',
  description: 'Free CLI for pre-commit code risk checks. Detect architectural risks, coupling, and incidents before they ship. MIT licensed, 2-5 second analysis.',
  keywords: 'code risk, pre-commit, AI code review, open source, CLI tool, architectural analysis',
  openGraph: {
    title: 'CodeRisk - Open Source AI Code Risk Assessment',
    description: 'Free CLI for pre-commit code risk checks.',
    url: 'https://coderisk.dev',
    siteName: 'CodeRisk',
    images: [
      {
        url: '/og-image.png', // Create this
        width: 1200,
        height: 630,
      },
    ],
  },
  twitter: {
    card: 'summary_large_image',
    title: 'CodeRisk - Open Source AI Code Risk Assessment',
    description: 'Free CLI for pre-commit code risk checks.',
    images: ['/og-image.png'],
  },
}
```

### 8. Visual Components

**Add Badge Components:**

Create reusable components for:
- MIT License badge (green, pill-shaped)
- GitHub stars badge (gray, with icon, fetch count dynamically)
- Version badge (if needed)

**Code Block Styling:**

Ensure code blocks have:
- Light gray background (`#F3F4F6`)
- Monospace font (Menlo, Monaco)
- Copy button (optional but nice)
- Syntax highlighting (optional)

**Platform Tabs (optional):**

For installation section, add tabs to switch between macOS, Linux, Windows commands. Use headless UI or Radix UI for accessibility.

### 9. Links & CTAs

**Ensure all CTAs work:**
- "Install CLI" → Scroll to installation section
- "View Docs" → Link to docs (placeholder `/docs` or external link)
- "View on GitHub" → `https://github.com/rohankatakam/coderisk-go`
- "Compare Plans" → `/pricing`
- "Start Free Trial" → `/pricing` (future: signup flow)
- "Read License" → `https://github.com/rohankatakam/coderisk-go/blob/main/LICENSE`
- "Contributing" → `https://github.com/rohankatakam/coderisk-go/blob/main/CONTRIBUTING.md`

### 10. Responsive Design

**Verify mobile responsiveness:**
- Hero headline should scale down on mobile (text-4xl → text-6xl)
- Feature grid should stack (3 columns → 1 column on mobile)
- Code blocks should scroll horizontally if needed
- Navigation should collapse to hamburger menu on mobile (if not already)

---

## Success Criteria

- [ ] Homepage hero emphasizes "Open Source"
- [ ] MIT License badge visible on homepage
- [ ] Installation instructions show all 4 methods
- [ ] Open source section explains what's free vs paid
- [ ] Cloud platform teaser leads to pricing page
- [ ] Pricing page created with 4 tiers
- [ ] Open source page explains open core model
- [ ] Navigation includes "Open Source" and "Pricing"
- [ ] Footer includes open source links and license notice
- [ ] Meta tags updated for SEO
- [ ] All links and CTAs work correctly
- [ ] Mobile responsive design verified
- [ ] Install script placeholder added to `public/`

---

## Key Files to Create/Modify

**Modified files:**
- `app/page.tsx` (homepage updates)
- `app/layout.tsx` (meta tags)
- `app/globals.css` (styling if needed)

**New files:**
- `app/pricing/page.tsx` (pricing page)
- `app/open-source/page.tsx` (open source page)
- `public/install.sh` (placeholder, will be replaced)
- `components/Badge.tsx` (badge components, optional)
- `components/CodeBlock.tsx` (code block with copy button, optional)

**Assets needed (future):**
- `/public/og-image.png` (Open Graph image for social sharing)

---

## Design Guidelines

Follow minimalist design from existing site:

**Colors:**
- Black: `#000000` (text, CTAs)
- White: `#FFFFFF` (background)
- Gray: `#6B7280` (secondary text)
- Green: `#10B981` (open source badge, success)
- Blue: `#3B82F6` (links)

**Typography:**
- Font: Inter or system UI
- Headings: Bold (700)
- Body: Regular (400)
- Code: Menlo, Monaco

**Spacing:**
- Sections: 96px vertical spacing (`py-24`)
- Content max width: 1024px (`max-w-4xl`)
- Grid gaps: 32px (`gap-8`)

See website_messaging.md Section "Visual Design Guidelines" for complete details.

---

## Notes

- Keep messaging developer-focused, not consumer marketing
- Emphasize "free forever" and "no lock-in"
- Make install commands easy to copy (one-click if possible)
- Use "open core" terminology, not "freemium"
- Highlight GitHub stars and community
- Ensure fast page load (<2 seconds)
- Optimize images (WebP format if possible)
- Test on Safari (macOS), Chrome (Linux), Edge (Windows)

---

## Reference Documentation

**Required Reading:**
- [website_messaging.md](website_messaging.md) - Complete content and strategy
- [open_core_strategy.md](../00-product/open_core_strategy.md) - Positioning
- [packaging_and_distribution.md](packaging_and_distribution.md) - Install methods

**Current Frontend:**
- Repository: `https://github.com/rohankatakam/coderisk-frontend`
- Live site: `https://coderisk.dev`

---

## Questions to Ask if Unclear

1. Should we add a docs page or link to external docs? (GitHub README or separate docs site?)
2. Do we have GitHub stars count API access? (Need for dynamic badge)
3. Should "Start Free Trial" link to pricing or a signup flow? (Suggest pricing for now)
4. Do we want testimonials? (Real user quotes - suggest placeholder for now)
5. Should we add a blog page? (Suggest defer to future)

---

**Good luck! This update will make CodeRisk's open source positioning crystal clear!**
