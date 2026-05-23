import { describe, expect, it, vi } from 'vitest';
import { createEChartController, type EChartHostWindow } from './EChartController';

describe('createEChartController', () => {
  it('updates, resizes, and disposes an ECharts instance safely', () => {
    const container = { nodeType: 1 } as HTMLElement;
    const chart = {
      dispose: vi.fn(),
      resize: vi.fn(),
      setOption: vi.fn(),
    };
    const engine = {
      init: vi.fn(() => chart),
    };
    const resizeListenerRef: { current?: () => void } = {};
    let observerCallback: ResizeObserverCallback = () => undefined;
    const observer = {
      disconnect: vi.fn(),
      observe: vi.fn(),
    };
    class ResizeObserverMock {
      constructor(callback: ResizeObserverCallback) {
        observerCallback = callback;
      }

      disconnect = observer.disconnect;
      observe = observer.observe;
    }

    const hostWindow: EChartHostWindow = {
      addEventListener: vi.fn((eventName, listener) => {
        if (eventName === 'resize') {
          resizeListenerRef.current = listener;
        }
      }),
      removeEventListener: vi.fn(),
      ResizeObserver: ResizeObserverMock,
    };

    const controller = createEChartController(container, engine, hostWindow);
    const option = { series: [] };

    controller.setOption(option);
    expect(resizeListenerRef.current).toBeDefined();
    resizeListenerRef.current?.();
    observerCallback([], observer as unknown as ResizeObserver);
    controller.dispose();
    controller.dispose();

    expect(engine.init).toHaveBeenCalledWith(container);
    expect(chart.setOption).toHaveBeenCalledWith(option, true);
    expect(observer.observe).toHaveBeenCalledWith(container);
    expect(chart.resize).toHaveBeenCalledTimes(2);
    expect(hostWindow.removeEventListener).toHaveBeenCalledWith('resize', resizeListenerRef.current);
    expect(observer.disconnect).toHaveBeenCalledTimes(1);
    expect(chart.dispose).toHaveBeenCalledTimes(1);
  });
});
