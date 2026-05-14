# Uikit Component API Reference

All components auto-registered via Nuxt; import is not required in `<template>`. Use the kebab-case tag name (e.g. `<lfx-button>`).

---

## LfxButton

**File:** `components/uikit/button/button.vue`

```vue
<lfx-button
  label="Click me"
  type="primary"        <!-- primary | secondary | tertiary | transparent | ghost | outline | nav -->
  button-style="pill"   <!-- rounded (default) | pill -->
  size="medium"         <!-- small | medium | large -->
  icon="box-dollar"     <!-- FA icon name, no fa- prefix -->
  icon-position="left"  <!-- left (default) | right -->
  :loading="false"
  :disabled="false"
/>
```

Use the default slot instead of `label` when you need rich content:

```vue
<lfx-button type="nav" button-style="pill" size="small">
  <lfx-icon name="ellipsis" type="light" :size="16" />
  More
</lfx-button>
```

---

## LfxIconButton

**File:** `components/uikit/icon-button/icon-button.vue`

```vue
<lfx-icon-button
  icon="grid-round"
  icon-type="solid"     <!-- light (default) | solid | regular | thin | duotone -->
  :icon-size="18"
  type="transparent"    <!-- default | transparent | primary | outline -->
  size="medium"         <!-- small | medium | large -->
  :disabled="false"
  aria-label="..."
/>
```

Use the default slot to override the icon (e.g. to place an avatar inside):

```vue
<lfx-icon-button type="transparent" size="medium" class="relative !bg-neutral-50">
  <img :src="user.avatarUrl" class="absolute inset-1.5 size-6 rounded-full object-cover" />
</lfx-icon-button>
```

---

## LfxIcon

**File:** `components/uikit/icon/icon.vue`

Renders a Font Awesome Pro icon (kit `d65f54d9ea` loaded globally).

```vue
<lfx-icon name="folder-heart" type="light" :size="16" />
```

| Prop   | Type   | Default | Notes                                     |
| ------ | ------ | ------- | ----------------------------------------- |
| `name` | string | —       | FA icon name without `fa-` prefix        |
| `type` | string | `light` | `light` \| `solid` \| `regular` \| `thin` \| `duotone` |
| `size` | number | `16`    | px value, converted to rem internally    |

---

## LfxDropdown

**File:** `components/uikit/dropdown/dropdown.vue`

Popper.js-based dropdown. Closes automatically when any item inside is clicked.

```vue
<lfx-dropdown
  v-model:visibility="isOpen"
  width="240px"
  placement="bottom-start"   <!-- any @popperjs/core Placement string -->
  :match-width="false"
  dropdown-class=""
  popover-class=""
>
  <template #trigger>
    <!-- anything clickable -->
  </template>

  <!-- default slot = panel content -->
  <lfx-dropdown-group-title>Section label</lfx-dropdown-group-title>
  <lfx-dropdown-item to="/path">...</lfx-dropdown-item>
  <lfx-dropdown-separator />
</lfx-dropdown>
```

Use `v-model:visibility` to track open state (e.g. to style the trigger when open).

---

## LfxDropdownItem

**File:** `components/uikit/dropdown/dropdown-item.vue`

Root element adapts: `NuxtLink` when `to` is set, `<a target="_blank">` when `href` is set, `<div>` otherwise.

```vue
<!-- Internal nav link -->
<lfx-dropdown-item to="/my-donations">
  <lfx-icon name="circle-dollar-to-slot" type="light" :size="16" />
  My donations
</lfx-dropdown-item>

<!-- External link -->
<lfx-dropdown-item href="https://example.com">External</lfx-dropdown-item>

<!-- Selection item (value-based) -->
<lfx-dropdown-item value="option-a" label="Option A" />
<lfx-dropdown-item value="option-a" :checkmark-before="true" label="Option A" />
```

Icons go **in the slot**, not as a prop.

---

## LfxDropdownGroupTitle / LfxDropdownSeparator

```vue
<lfx-dropdown-group-title>Solutions</lfx-dropdown-group-title>
<lfx-dropdown-separator />
```

---

## LfxDropdownSelect / LfxDropdownSelector

Higher-level select built on top of `LfxDropdown`. Use for value-selection dropdowns where one item is "selected".

```vue
<lfx-dropdown-select v-model="selectedValue" width="200px">
  <lfx-dropdown-item value="a" label="Option A" />
  <lfx-dropdown-item value="b" label="Option B" />
</lfx-dropdown-select>
```

---

## LfxInput

**File:** `components/uikit/input/input.vue`

```vue
<lfx-input v-model="value" placeholder="Search..." :disabled="false" />
```

---

## LfxTextarea

```vue
<lfx-textarea v-model="value" placeholder="..." :rows="4" />
```

---

## LfxSelect + LfxOption

```vue
<lfx-select v-model="selected">
  <lfx-option value="a">Option A</lfx-option>
  <lfx-option value="b">Option B</lfx-option>
</lfx-select>
```

---

## LfxCheckbox

```vue
<lfx-checkbox v-model="checked" label="Accept terms" :disabled="false" />
```

---

## LfxRadio

```vue
<lfx-radio v-model="selected" value="option-a" label="Option A" />
```

---

## LfxToggle

```vue
<lfx-toggle v-model="enabled" label="Enable notifications" />
```

---

## LfxField + LfxFieldMessage + LfxFieldMessages

Wraps form inputs with label and validation messages.

```vue
<lfx-field label="Email" :required="true">
  <lfx-input v-model="email" />
  <lfx-field-messages :messages="v$.email.$errors" />
</lfx-field>
```

---

## LfxModal

```vue
<lfx-modal v-model:visible="showModal" title="Confirm action">
  <p>Are you sure?</p>
  <template #footer>
    <lfx-button label="Cancel" type="ghost" @click="showModal = false" />
    <lfx-button label="Confirm" @click="confirm" />
  </template>
</lfx-modal>
```

---

## LfxDrawer

```vue
<lfx-drawer
  v-model="showDrawer"
  position="right"
  width="37.5rem"
  :close-function="() => true"
  :hide-close-button="false"
>
  <!-- default slot receives { close } -->
  <template #default="{ close }">
    <!-- drawer content -->
  </template>
</lfx-drawer>
```

| Prop               | Type                        | Default      | Notes                                              |
| ------------------ | --------------------------- | ------------ | -------------------------------------------------- |
| `modelValue`       | `boolean`                   | —            | v-model open/closed state                          |
| `position`         | `'left' \| 'right' \| 'bottom'` | `'right'`  | Slide direction                                    |
| `width`            | `string`                    | `'37.5rem'`  | max-width for left/right positions                 |
| `height`           | `string`                    | `'85vh'`     | max-height for bottom position                     |
| `closeFunction`    | `() => boolean`             | `() => true` | Return false to veto close (e.g. unsaved changes)  |
| `hideCloseButton`  | `boolean`                   | `false`      | Hide the built-in × button to add your own         |

The `bottom` position adds `rounded-t-2xl` to the panel and fills the full viewport width.

---

## LfxTooltip

```vue
<lfx-tooltip content="Helpful hint">
  <lfx-icon-button icon="circle-info" />
</lfx-tooltip>
```

---

## LfxPopover

Low-level Popper.js wrapper used by `LfxDropdown`. Prefer `LfxDropdown` unless you need custom panel layout.

```vue
<lfx-popover v-model:visibility="open" placement="bottom-start">
  <template #trigger><button>Open</button></template>
  <template #content>...</template>
</lfx-popover>
```

---

## LfxAvatar + LfxAvatarGroup

```vue
<lfx-avatar :src="user.avatarUrl" :alt="user.name" size="medium" />
<lfx-avatar-group :users="[{ name, avatarUrl }]" :max="5" />
```

---

## LfxCard

```vue
<lfx-card>
  <!-- card content -->
</lfx-card>
```

---

## LfxChip

```vue
<lfx-chip label="Active" type="success" />
```

---

## LfxTag

```vue
<lfx-tag label="Open Source" />
```

---

## LfxProgressBar

```vue
<lfx-progress-bar :value="60" :max="100" />
```

---

## LfxSpinner

```vue
<lfx-spinner />
```

---

## LfxSkeleton

```vue
<lfx-skeleton class="h-4 w-32 rounded" />
```

---

## LfxTabs + LfxTabsPanels

```vue
<lfx-tabs v-model="activeTab" :tabs="[{ label: 'Overview', value: 'overview' }]" />
<lfx-tabs-panels v-model="activeTab">
  <template #overview>...</template>
</lfx-tabs-panels>
```

---

## LfxAccordion + LfxAccordionItem

```vue
<lfx-accordion>
  <lfx-accordion-item title="Section 1">Content</lfx-accordion-item>
</lfx-accordion>
```

---

## LfxTable

```vue
<lfx-table :columns="cols" :rows="rows" />
```

---

## LfxScrollView + LfxScrollArea + LfxScrollableShadow

Use when content overflows and needs a styled scrollable region.

---

## LfxBack

Back/breadcrumb navigation link.

```vue
<lfx-back to="/initiatives">Back to Initiatives</lfx-back>
```

---

## LfxOrganizationLogo

```vue
<lfx-organization-logo :src="org.logoUrl" :alt="org.name" />
```

---

## LfxMenuButton

Icon button that opens an attached dropdown menu.

---

## LfxShare

Social share UI element.

---

## LfxToast

Programmatic toast notifications — check the component for the composable API.

---

## LfxDatepicker

```vue
<lfx-datepicker v-model="date" />
```

---

## LfxDropdownSearch

Search input styled for use inside a `LfxDropdown` panel.

```vue
<lfx-dropdown v-model:visibility="open" width="280px">
  <template #trigger>...</template>
  <lfx-dropdown-search v-model="query" placeholder="Search..." />
  <lfx-dropdown-item v-for="item in filtered" :key="item.value" :value="item.value">
    {{ item.label }}
  </lfx-dropdown-item>
</lfx-dropdown>
```

---

## LfxDropdownSubmenu

Nested submenu inside a dropdown.

```vue
<lfx-dropdown-submenu label="More options">
  <lfx-dropdown-item to="/foo">Foo</lfx-dropdown-item>
</lfx-dropdown-submenu>
```
