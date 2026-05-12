// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

export interface FooterMenuLink {
  name: string;
  link: string;
}

export interface FooterMenuSection {
  title: string;
  links: FooterMenuLink[];
}

export const lfxFooterMenu: FooterMenuSection[] = [
  {
    title: 'Platform',
    links: [
      { name: 'Explore initiatives', link: '/initiatives' },
      { name: 'Statistics', link: '/statistics' },
      { name: 'Start a Fundraise', link: '/start-fundraise' },
      { name: 'About', link: '/about' },
    ],
  },
  {
    title: 'Solutions',
    links: [
      { name: 'For Projects', link: '/for-projects' },
      { name: 'For Companies', link: '/for-companies' },
    ],
  },
  {
    title: 'The Linux Foundation',
    links: [
      { name: 'LFX Self Serve', link: 'https://lfx.linuxfoundation.org' },
      { name: 'LFX Insights', link: 'https://insights.lfx.linuxfoundation.org' },
      { name: 'LFX Mentorship', link: 'https://mentorship.lfx.linuxfoundation.org' },
      { name: 'About the LF', link: 'https://www.linuxfoundation.org/about' },
    ],
  },
];
