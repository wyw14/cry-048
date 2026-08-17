import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import { defineComponent, h } from 'vue'

describe('sanity', () => {
  it('mounts a trivial Vue component', () => {
    const Hello = defineComponent({
      name: 'Hello',
      render() {
        return h('div', { class: 'hi' }, 'hello world')
      },
    })
    const w = mount(Hello)
    expect(w.find('.hi').text()).toBe('hello world')
  })
})
