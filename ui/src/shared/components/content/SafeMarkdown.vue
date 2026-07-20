<script>
import { defineComponent, h } from 'vue'
import MarkdownIt from 'markdown-it'

const markdown = new MarkdownIt({
  html: false,
  linkify: true,
  breaks: true,
})

const allowedTags = new Set([
  'a',
  'blockquote',
  'code',
  'del',
  'em',
  'h1',
  'h2',
  'h3',
  'h4',
  'h5',
  'h6',
  'li',
  'ol',
  'p',
  'strong',
  'table',
  'tbody',
  'td',
  'th',
  'thead',
  'tr',
  'ul',
])

function tokenTree(tokens) {
  const root = []
  const stack = [root]

  for (const token of tokens) {
    if (token.nesting === -1) {
      stack.pop()
      continue
    }

    const node = {
      token,
      children: token.children ? tokenTree(token.children) : [],
    }
    stack.at(-1).push(node)

    if (token.nesting === 1) stack.push(node.children)
  }

  return root
}

function safeLink(value) {
  const normalized = markdown.normalizeLink(value)
  return markdown.validateLink(normalized) ? normalized : null
}

function tokenAttributes(token) {
  if (token.tag === 'a') {
    const href = safeLink(token.attrGet('href') || '')
    return {
      ...(href ? { href } : {}),
      ...(token.attrGet('title') ? { title: token.attrGet('title') } : {}),
    }
  }

  if (token.tag === 'ol' && token.attrGet('start')) {
    return { start: token.attrGet('start') }
  }

  if (token.tag === 'code' && /^language-[\w-]+$/.test(token.attrGet('class') || '')) {
    return { class: token.attrGet('class') }
  }

  return {}
}

function renderNodes(nodes, prefix = 'markdown') {
  return nodes.map((node, index) => renderNode(node, `${prefix}-${index}`))
}

function renderNode({ token, children }, key) {
  if (token.type === 'text') return token.content
  if (token.type === 'softbreak' || token.type === 'hardbreak') return h('br', { key })
  if (token.type === 'code_inline') return h('code', { key }, token.content)
  if (token.type === 'fence' || token.type === 'code_block') {
    return h('pre', { key }, [h('code', tokenAttributes(token), token.content)])
  }
  if (token.type === 'hr') return h('hr', { key })
  if (token.type === 'image') {
    const src = safeLink(token.attrGet('src') || '')
    return src
      ? h('img', {
          key,
          src,
          alt: token.content,
          ...(token.attrGet('title') ? { title: token.attrGet('title') } : {}),
        })
      : token.content
  }
  if (token.type === 'inline') return renderNodes(children, key)
  if (!allowedTags.has(token.tag)) return token.content || renderNodes(children, key)

  return h(token.tag, { key, ...tokenAttributes(token) }, renderNodes(children, key))
}

export default defineComponent({
  name: 'SafeMarkdown',
  inheritAttrs: false,
  props: {
    source: {
      type: String,
      default: '',
    },
    fallback: {
      type: String,
      default: '',
    },
  },
  setup(props, { attrs }) {
    return () => {
      const source = props.source.trim()
      const content = source
        ? renderNodes(tokenTree(markdown.parse(source, {})))
        : [h('p', props.fallback)]
      return h('div', attrs, content)
    }
  },
})
</script>
