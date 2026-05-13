// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import jenkinsLogo from '~/assets/images/jenkins_logo.svg';

export interface Testimonial {
  id: string;
  logoSrc: string;
  logoAlt: string;
  quote: string;
  authorName: string;
  authorTitle: string;
}

export const testimonials: Testimonial[] = [
  {
    id: 'jenkins',
    logoSrc: jenkinsLogo,
    logoAlt: 'Jenkins',
    quote:
      'We use LFX Crowdfunding to receive donations and handle project expenses. It provides all key features for working with individual and company donors, and it helps us to facilitate Jenkins evolution by funding outreach programs and infrastructure costs. We have also run our first independent mentoring project with the Mentorship portal. We are looking forward to making LFX the primary crowdfunding platform!',
    authorName: 'Oleg Nenashev',
    authorTitle: 'Jenkins Core Maintainer and Board Member',
  },
];
