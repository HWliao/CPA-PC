import { useEffect, useRef, type CSSProperties } from 'react';
import type { EChartsCoreOption } from 'echarts/core';
import { createEChartController, type EChartController } from './EChartController';

export interface EChartPanelProps {
  ariaLabel: string;
  className?: string;
  option: EChartsCoreOption;
  style?: CSSProperties;
}

export function EChartPanel({ ariaLabel, className, option, style }: EChartPanelProps) {
  const containerRef = useRef<HTMLDivElement>(null);
  const controllerRef = useRef<EChartController | null>(null);

  useEffect(() => {
    if (!containerRef.current) return undefined;
    const controller = createEChartController(containerRef.current);
    controllerRef.current = controller;
    return () => {
      controller.dispose();
      controllerRef.current = null;
    };
  }, []);

  useEffect(() => {
    controllerRef.current?.setOption(option);
  }, [option]);

  return <div ref={containerRef} role="img" aria-label={ariaLabel} className={className} style={style} />;
}
