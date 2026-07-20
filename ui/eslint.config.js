import js from '@eslint/js'
import globals from 'globals'
import pluginVue from 'eslint-plugin-vue'
import tailwindcss from 'eslint-plugin-tailwindcss'
import prettierConfig from '@vue/eslint-config-prettier'

export default [
  js.configs.recommended,
  ...pluginVue.configs['flat/recommended'],
  prettierConfig,
  {
    plugins: {
      tailwindcss,
    },
    languageOptions: {
      globals: {
        ...globals.browser,
      },
    },
    rules: {
      'tailwindcss/no-contradicting-classname': 'error',
      'vue/multi-word-component-names': 'off',
    },
    settings: {
      tailwindcss: {
        cssConfigPath: './src/styles.css',
      },
    },
  },
]
