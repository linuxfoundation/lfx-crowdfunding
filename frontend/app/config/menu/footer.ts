// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import { useFundraiseDrawerStore } from '~/components/modules/fundraise/store/fundraise-drawer.store';
import { authState, login } from '~/composables/useAuth';
import { AppRoute } from '~/config/routes';
import type { FooterMenuSection } from '~/types/footer.types';

export const lfxFooterMenu: FooterMenuSection[] = [
  {
    title: 'Platform',
    links: [
      { name: 'Explore initiatives', link: AppRoute.Initiatives },
      { name: 'Statistics', link: AppRoute.Statistics },
      {
        name: 'Start a Fundraise',
        action: () => {
          if (!authState.value.isAuthenticated) {
            login();
          } else {
            useFundraiseDrawerStore().openFundraiseDrawer();
          }
        },
      },
      { name: 'About', link: AppRoute.About },
      { name: 'Documentation', link: AppRoute.Docs },
      { name: 'Contact support', intercom: true },
    ],
  },
  {
    title: 'Solutions',
    links: [
      { name: 'For Projects', link: AppRoute.ForProjects },
      { name: 'For Companies', link: AppRoute.ForCompanies },
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
