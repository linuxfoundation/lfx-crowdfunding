// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

export interface DonutChartColor {
  from: string;
  to: string;
}

export type DonutChartColorProp = string | DonutChartColor;

export const isGradient = (color: DonutChartColorProp): color is DonutChartColor =>
  typeof color === 'object' && 'from' in color;

export interface DonutChartSegment {
  /** 0–100 percentage for this segment */
  value: number;
  /** Solid color string OR { from, to } gradient object */
  color: DonutChartColorProp;
}

export interface ResolvedSegment {
  dash: number;
  startAngle: number;
  stroke: string;
}

export interface GradientMeta {
  id: string;
  color: DonutChartColorProp;
}
