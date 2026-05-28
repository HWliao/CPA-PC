import { describe, expect, it } from 'vitest';
import en from './locales/en.json';
import ru from './locales/ru.json';
import zhCN from './locales/zh-CN.json';
import zhTW from './locales/zh-TW.json';

describe('model price labels', () => {
  it('uses input/output/input-cache terminology in primary locales', () => {
    expect(en.usage_stats.model_price_prompt).toBe('Input price');
    expect(en.usage_stats.model_price_completion).toBe('Output price');
    expect(en.usage_stats.model_price_cache).toBe('Input cache price');

    expect(zhCN.usage_stats.model_price_prompt).toBe('输入价格');
    expect(zhCN.usage_stats.model_price_completion).toBe('输出价格');
    expect(zhCN.usage_stats.model_price_cache).toBe('输入缓存价格');
  });

  it('removes prompt and completion terminology from secondary locales', () => {
    expect(zhTW.usage_stats.model_price_prompt).not.toContain('提示');
    expect(zhTW.usage_stats.model_price_completion).not.toContain('補全');
    expect(ru.usage_stats.model_price_prompt.toLowerCase()).not.toContain('prompt');
    expect(ru.usage_stats.model_price_completion.toLowerCase()).not.toContain('completion');
  });
});
