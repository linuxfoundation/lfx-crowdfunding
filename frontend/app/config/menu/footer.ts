// Copyright (c) 2025 The Linux Foundation and each contributor.
// SPDX-License-Identifier: MIT

export interface FooterMenuLink {
  link?: string;
  name: string;
}

export interface FooterMenuSection {
  title: string;
  links: FooterMenuLink[];
}

export const lfxFooterMenu: FooterMenuSection[] = [
  {
    title: 'LFX Crowdfunding',
    links: [
      { name: 'Projects', link: '/campaigns?type=project' },
      { name: 'Mentorships', link: '/campaigns?type=mentorship' },
      { name: 'Events', link: '/campaigns?type=event' },
      { name: 'General Funds', link: '/campaigns?type=general_fund' },
      { name: 'Changelog', link: 'https://changelog.lfx.dev/?product=crowdfunding' },
      { name: 'Roadmap', link: 'https://changelog.lfx.dev/roadmap?product=crowdfunding' },
    ],
  },
  {
    title: 'Resources',
    links: [
      { name: 'Documentation', link: 'https://docs.linuxfoundation.org/lfx/crowdfunding' },
      { name: 'LFX Community', link: 'https://community.lfx.dev' },
      { name: 'Linux Foundation', link: 'https://www.linuxfoundation.org' },
    ],
  },
  {
    title: 'Other LFX Tools',
    links: [
      { name: 'Insights', link: 'https://insights.lfx.linuxfoundation.org' },
      { name: 'Mentorship', link: 'https://mentorship.lfx.linuxfoundation.org' },
      { name: 'EasyCLA', link: 'https://easycla.lfx.linuxfoundation.org' },
      { name: 'Security', link: 'https://security.lfx.linuxfoundation.org' },
    ],
  },
];
