import { defineConfig, presetAttributify, presetIcons, presetUno } from 'unocss'

export default defineConfig({
  presets: [
    presetUno(),
    presetAttributify(),
    presetIcons({
      scale: 1.1,
      warn: true,
    }),
  ],
  theme: {
    colors: {
      ink: {
        950: '#101418',
        900: '#182029',
        700: '#344253',
        500: '#647184',
        300: '#a7b0bd',
      },
      brand: {
        600: '#2563eb',
        500: '#3174f1',
      },
    },
  },
  shortcuts: {
    'page-shell': 'min-h-screen bg-[#f6f8fb] text-ink-900',
    'panel': 'rounded-2 border border-[#e4e9f1] bg-white shadow-[0_1px_2px_rgba(15,23,42,0.04)]',
    'field': 'h-9 rounded-1.5 border border-[#d7deea] bg-white px-3 text-13px text-ink-900 outline-none transition focus:border-brand-500 focus:ring-3 focus:ring-blue-100',
    'btn': 'inline-flex h-9 items-center justify-center gap-2 rounded-1.5 px-3 text-13px font-medium transition disabled:cursor-not-allowed disabled:opacity-50',
    'btn-primary': 'btn bg-brand-600 text-white hover:bg-brand-500',
    'btn-soft': 'btn border border-[#d7deea] bg-white text-ink-700 hover:bg-[#f7f9fc]',
    'btn-danger': 'btn bg-[#dc2626] text-white hover:bg-[#b91c1c]',
    'th': 'bg-[#f8fafc] px-3 py-2.5 text-left text-12px font-semibold uppercase tracking-wide text-ink-500 whitespace-nowrap',
    'td': 'border-t border-[#edf1f7] px-3 py-2.5 text-13px text-ink-700 whitespace-nowrap align-top',
  },
})
