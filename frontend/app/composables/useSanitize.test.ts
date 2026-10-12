// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT
import { describe, expect, it } from 'vitest';
import { useSanitize } from './useSanitize';

const { renderDescription, sanitize } = useSanitize();

describe('renderDescription', () => {
  it('keeps editor formatting', () => {
    const html =
      '<h2>Title</h2><p><strong>b</strong> <em>i</em> <u>u</u> <s>s</s></p><ul><li>x</li></ul><blockquote>q</blockquote>';
    expect(renderDescription(html)).toBe(html);
  });

  it('removes forms, styles, images and inline styles', () => {
    const out = renderDescription(
      '<style>body{display:none}</style><form action="https://evil.example"><input name="pw"><button>Sign in</button></form>' +
        '<img src="https://evil.example/p.gif"><div style="position:fixed;inset:0">overlay</div><p class="x">ok</p>',
    );
    expect(out).not.toMatch(/<(style|form|input|button|img|div)\b|style=|class=|evil\.example/i);
    expect(out).toContain('<p>ok</p>');
  });

  it('hardens links and drops non-http(s)/mailto URLs', () => {
    expect(renderDescription('<a href="https://lfx.dev">x</a>')).toBe(
      '<a href="https://lfx.dev" target="_blank" rel="noopener noreferrer nofollow">x</a>',
    );
    expect(renderDescription('<a href="mailto:a@b.c">m</a>')).toContain('href="mailto:a@b.c"');
    expect(renderDescription('<a href="javascript:alert(1)">x</a>')).toBe(
      '<a target="_blank" rel="noopener noreferrer nofollow">x</a>',
    );
  });

  it('still converts plain text to paragraphs', () => {
    expect(renderDescription('a\nb\n\nc')).toBe('<p>a<br>b</p><p>c</p>');
  });

  it('leaves sanitize() on the default profile without the link hook', () => {
    expect(sanitize('<a href="/x" class="c">x</a>')).toBe('<a href="/x" class="c">x</a>');
  });
});
