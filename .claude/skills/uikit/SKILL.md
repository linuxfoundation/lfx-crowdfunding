---
name: uikit
description: Use when building any UI element in the crowdfunding frontend — covers when to use existing uikit components, how to extend them, and when to ask before creating new ones
license: MIT
---

# LFX Uikit

Reference for using and extending the LFX uikit component library in `frontend/app/components/uikit/`.

## Rules — always apply these

1. **Always check the uikit first.** Before writing any UI element (button, icon, dropdown, input, modal, etc.) look up the relevant component below. If it exists, use it.

2. **Extend, don't copy.** If a component almost fits but lacks a variant or prop, add it to the component and its types file rather than building a one-off styled element.

3. **Ask before creating new components.** If no uikit component covers the need, describe what it would do and why nothing existing works — then wait for the user's decision before building.

4. **No raw HTML equivalents.** Never write:
   - `<button>` → use `<lfx-button>` or `<lfx-icon-button>`
   - `<i class="fa-...">` → use `<lfx-icon>`
   - Custom dropdown panels with `v-show` + manual listeners → use `<lfx-dropdown>`
   - Custom spinner/skeleton HTML → use `<lfx-spinner>` / `<lfx-skeleton>`

5. **Never define types or interfaces inside `.vue` files.** All shared types belong in `frontend/app/types/`. Name the file after the domain (e.g. `fundraise.types.ts`). Import with `import type { ... } from '~/types/...'`. A component may re-export a type from the types file (`export type { Foo };`) only if external callers currently rely on that import path, but the canonical definition must live in `app/types/`.

## Quick reference

| Need                          | Component(s)                                               |
| ----------------------------- | ---------------------------------------------------------- |
| Clickable button              | `LfxButton`                                                |
| Icon-only button              | `LfxIconButton`                                            |
| Font Awesome icon             | `LfxIcon`                                                  |
| Dropdown / menu               | `LfxDropdown` + `LfxDropdownItem` + helpers               |
| Text input                    | `LfxInput`                                                 |
| Textarea                      | `LfxTextarea`                                              |
| Select / combobox             | `LfxSelect` + `LfxDropdownSelect` + `LfxDropdownSelector` |
| Checkbox                      | `LfxCheckbox`                                              |
| Radio button                  | `LfxRadio`                                                 |
| Toggle / switch               | `LfxToggle`                                                |
| Date picker                   | `LfxDatepicker`                                            |
| Form field wrapper            | `LfxField` + `LfxFieldMessage` + `LfxFieldMessages`       |
| Modal / dialog                | `LfxModal`                                                 |
| Drawer / side panel           | `LfxDrawer`                                                |
| Tooltip                       | `LfxTooltip`                                               |
| Popover (raw)                 | `LfxPopover`                                               |
| Avatar                        | `LfxAvatar` + `LfxAvatarGroup`                            |
| Card                          | `LfxCard`                                                  |
| Chip / badge                  | `LfxChip`                                                  |
| Tag                           | `LfxTag`                                                   |
| Progress bar                  | `LfxProgressBar`                                           |
| Spinner / loading             | `LfxSpinner`                                               |
| Skeleton loader               | `LfxSkeleton`                                              |
| Tabs                          | `LfxTabs` + `LfxTabsPanels`                               |
| Accordion                     | `LfxAccordion` + `LfxAccordionItem`                       |
| Carousel                      | `LfxCarousel` + `LfxCarouselNavigation`                   |
| Table                         | `LfxTable`                                                 |
| Scrollable container          | `LfxScrollView` + `LfxScrollArea` + `LfxScrollableShadow` |
| Back / breadcrumb nav         | `LfxBack`                                                  |
| Organization logo             | `LfxOrganizationLogo`                                      |
| Menu button (icon+label+menu) | `LfxMenuButton`                                            |
| Share                         | `LfxShare`                                                 |
| Toast notification            | `LfxToast`                                                 |

## Loading the detailed reference

- [ ] [references/components.md](references/components.md) — props, slots, and usage examples for every uikit component
