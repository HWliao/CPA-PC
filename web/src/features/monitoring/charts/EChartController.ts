import { LineChart } from 'echarts/charts';
import { DataZoomComponent, GridComponent, LegendComponent, TitleComponent, TooltipComponent } from 'echarts/components';
import { init, use as registerEChartsModules, type EChartsCoreOption } from 'echarts/core';
import { CanvasRenderer } from 'echarts/renderers';

registerEChartsModules([
  LineChart,
  GridComponent,
  LegendComponent,
  TitleComponent,
  TooltipComponent,
  DataZoomComponent,
  CanvasRenderer,
]);

export type EChartInstance = {
  dispose: () => void;
  resize: () => void;
  setOption: (option: EChartsCoreOption, notMerge?: boolean) => void;
};

export type EChartEngine = {
  init: (container: HTMLElement) => EChartInstance;
};

export type EChartHostWindow = {
  addEventListener: (eventName: 'resize', listener: () => void) => void;
  removeEventListener: (eventName: 'resize', listener: () => void) => void;
  ResizeObserver?: new (callback: ResizeObserverCallback) => {
    disconnect: () => void;
    observe: (target: Element) => void;
  };
};

export type EChartController = {
  dispose: () => void;
  setOption: (option: EChartsCoreOption) => void;
};

const defaultEngine: EChartEngine = {
  init: (container) => init(container),
};

const getHostWindow = (): EChartHostWindow | undefined => {
  if (typeof window === 'undefined') return undefined;
  return window;
};

export function createEChartController(
  container: HTMLElement,
  engine: EChartEngine = defaultEngine,
  hostWindow: EChartHostWindow | undefined = getHostWindow()
): EChartController {
  const chart = engine.init(container);
  let disposed = false;
  const resize = () => {
    if (!disposed) {
      chart.resize();
    }
  };
  const resizeObserver = hostWindow?.ResizeObserver ? new hostWindow.ResizeObserver(resize) : null;

  resizeObserver?.observe(container);
  hostWindow?.addEventListener('resize', resize);

  return {
    setOption: (option) => {
      if (!disposed) {
        chart.setOption(option, true);
      }
    },
    dispose: () => {
      if (disposed) return;
      disposed = true;
      hostWindow?.removeEventListener('resize', resize);
      resizeObserver?.disconnect();
      chart.dispose();
    },
  };
}
