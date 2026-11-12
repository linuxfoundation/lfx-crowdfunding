// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

// Testimonial logos live in public/ and are referenced by fixed path. Importing them here
// makes Vite emit an absolute URL in the client bundle but a relative one on the server,
// so the bound :src fails hydration on every direct load of /for-projects.
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
    logoSrc: '/images/testimonials/jenkins_logo.svg',
    logoAlt: 'Jenkins',
    quote:
      'We use LFX Crowdfunding to receive donations and handle project expenses. It provides all key features for working with individual and company donors, and it helps us to facilitate Jenkins evolution by funding outreach programs and infrastructure costs. We have also run our first independent mentoring project with the Mentorship portal. We are looking forward to making LFX the primary crowdfunding platform!',
    authorName: 'Oleg Nenashev',
    authorTitle: 'Jenkins Core Maintainer and Board Member',
  },
];
