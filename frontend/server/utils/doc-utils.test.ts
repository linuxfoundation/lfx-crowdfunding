// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import { resolve, join } from 'node:path';
import { describe, it, expect } from 'vitest';
import { rewriteDocLink, parseFrontmatter, formatDate, toTitleCase } from './doc-utils';

// ── rewriteDocLink ─────────────────────────────────────────────────────────

describe('rewriteDocLink', () => {
  describe('pass-through cases', () => {
    it('leaves anchor links unchanged', () => {
      expect(rewriteDocLink('#overview', 'initiatives')).toBe('#overview');
    });

    it('leaves root-relative paths unchanged', () => {
      expect(rewriteDocLink('/about', 'initiatives')).toBe('/about');
    });

    it('leaves http URLs unchanged', () => {
      expect(rewriteDocLink('http://example.com', 'initiatives')).toBe('http://example.com');
    });

    it('leaves https URLs unchanged', () => {
      expect(rewriteDocLink('https://example.com/page', 'donations')).toBe(
        'https://example.com/page',
      );
    });

    it('leaves mailto: links unchanged', () => {
      expect(rewriteDocLink('mailto:support@example.com', 'initiatives')).toBe(
        'mailto:support@example.com',
      );
    });

    it('leaves tel: links unchanged', () => {
      expect(rewriteDocLink('tel:+15551234567', 'initiatives')).toBe('tel:+15551234567');
    });
  });

  describe('./ same-directory links', () => {
    it('rewrites ./child/ inside a section', () => {
      expect(rewriteDocLink('./make-donation/', 'donations')).toBe('/docs/donations/make-donation');
    });

    it('rewrites ./child with no trailing slash', () => {
      expect(rewriteDocLink('./make-donation', 'donations')).toBe('/docs/donations/make-donation');
    });

    it('rewrites ./child.md stripping the extension', () => {
      expect(rewriteDocLink('./make-donation.md', 'donations')).toBe(
        '/docs/donations/make-donation',
      );
    });
  });

  describe('../ parent-relative links', () => {
    it('rewrites single ../ to a sibling of the parent', () => {
      // From donations/make-donation, ../ pops make-donation → donations/initiatives
      expect(rewriteDocLink('../initiatives/', 'donations/make-donation')).toBe(
        '/docs/donations/initiatives',
      );
    });

    it('rewrites double ../../ correctly (two levels up)', () => {
      // slug = initiatives/manage-initiative → slugDir = initiatives/manage-initiative (index)
      // ../../reimbursements/ should resolve to /docs/reimbursements
      expect(rewriteDocLink('../../reimbursements/', 'initiatives/manage-initiative')).toBe(
        '/docs/reimbursements',
      );
    });

    it('handles ../ from a top-level section (no parent)', () => {
      // Going above the root — parts stack empties gracefully
      expect(rewriteDocLink('../other/', 'initiatives')).toBe('/docs/other');
    });
  });

  describe('bare relative links (no prefix)', () => {
    it('treats bare names as same-directory', () => {
      expect(rewriteDocLink('make-donation', 'donations')).toBe('/docs/donations/make-donation');
    });
  });

  describe('cleaning', () => {
    it('strips trailing /index.md', () => {
      expect(rewriteDocLink('./sub/index.md', 'initiatives')).toBe('/docs/initiatives/sub');
    });

    it('strips trailing slash', () => {
      expect(rewriteDocLink('./sub/', 'initiatives')).toBe('/docs/initiatives/sub');
    });
  });
});

// ── path traversal guard ────────────────────────────────────────────────────
// This reproduces the guard logic from [...slug].get.ts so the safety property
// is covered by a fast pure test (no filesystem access needed).

function isSlugSafe(slug: string, docsDir: string): boolean {
  const safe = resolve(docsDir);
  return [resolve(join(safe, `${slug}.md`)), resolve(join(safe, slug, 'index.md'))].every((c) =>
    c.startsWith(safe + '/'),
  );
}

describe('slug path traversal guard', () => {
  const docsDir = '/srv/docs/user';

  it('allows a simple top-level slug', () => {
    expect(isSlugSafe('initiatives', docsDir)).toBe(true);
  });

  it('allows a nested slug', () => {
    expect(isSlugSafe('initiatives/create-initiative', docsDir)).toBe(true);
  });

  it('rejects a slug containing ../', () => {
    expect(isSlugSafe('../../etc/passwd', docsDir)).toBe(false);
  });

  it('rejects a slug that escapes docsDir by one level', () => {
    expect(isSlugSafe('../other-dir/secret', docsDir)).toBe(false);
  });
});

// ── parseFrontmatter ────────────────────────────────────────────────────────

describe('parseFrontmatter', () => {
  it('extracts YAML data and body', () => {
    const raw = `---\ntitle: Hello\n---\nBody text`;
    const { data, content } = parseFrontmatter(raw);
    expect(data.title).toBe('Hello');
    expect(content).toBe('Body text');
  });

  it('returns empty data and full text when there is no frontmatter', () => {
    const raw = 'Just plain content';
    const { data, content } = parseFrontmatter(raw);
    expect(data).toEqual({});
    expect(content).toBe('Just plain content');
  });
});

// ── formatDate ──────────────────────────────────────────────────────────────

describe('formatDate', () => {
  it('formats a Date object to YYYY-MM-DD', () => {
    expect(formatDate(new Date('2026-06-16T12:00:00Z'))).toBe('2026-06-16');
  });

  it('passes through a string date', () => {
    expect(formatDate('2026-01-01')).toBe('2026-01-01');
  });
});

// ── toTitleCase ─────────────────────────────────────────────────────────────

describe('toTitleCase', () => {
  it('capitalises each hyphen-separated word', () => {
    expect(toTitleCase('getting-started')).toBe('Getting Started');
  });

  it('handles a single word', () => {
    expect(toTitleCase('donations')).toBe('Donations');
  });
});
