<script lang="ts">
  import { onMount } from 'svelte'
  import { groups, exampleCount } from './lib/examples'
  import Example from './components/Example.svelte'

  const checklistURL = 'https://github.com/bencode/sumi/blob/main/docs/compatibility.md'

  let theme = $state<'system' | 'light' | 'dark'>('system')
  let systemDark = $state(false)
  let navigationHeight = $state(56)
  const dark = $derived(theme === 'dark' || (theme === 'system' && systemDark))

  onMount(() => {
    const preference = window.matchMedia('(prefers-color-scheme: dark)')
    systemDark = preference.matches
    const changed = (event: MediaQueryListEvent) => { systemDark = event.matches }
    preference.addEventListener('change', changed)
    return () => preference.removeEventListener('change', changed)
  })

  $effect(() => { document.documentElement.dataset.theme = dark ? 'dark' : 'light' })
  $effect(() => { document.documentElement.style.setProperty('--category-nav-height', `${navigationHeight}px`) })
</script>

<a class="skip-link" href="#examples">Skip to examples</a>
<header class="site-header">
  <a class="wordmark" href="#top" aria-label="sumi, top of page">墨 <span>sumi</span></a>
  <div class="header-actions">
    <a href={checklistURL} target="_blank" rel="noreferrer">Feature checklist <span aria-hidden="true">↗</span></a>
    <a href="https://github.com/bencode/sumi">GitHub <span aria-hidden="true">↗</span></a>
    <label class="theme-control">Theme
      <select bind:value={theme} aria-label="Color theme">
        <option value="system">System</option>
        <option value="light">Light</option>
        <option value="dark">Dark</option>
      </select>
    </label>
  </div>
</header>
<main id="top">
  <div class="intro">
    <p class="eyebrow">A diagram, from source to scene</p>
    <h1>Mermaid, drawn in Go.</h1>
    <p class="lede">Explore what sumi draws. Change the source of any example and watch Go turn it into a diagram.</p>
    <p class="build">{exampleCount} editable examples · {groups.length} diagram types · Light &amp; dark</p>
    <p class="intro-note">Previews update after you stop typing. Your edits stay in this page; reload to start fresh.</p>
  </div>
  <nav class="category-nav" aria-label="Diagram types" bind:offsetHeight={navigationHeight}>
    {#each groups as group (group.id)}<a href={`#${group.id}`}>{group.title}</a>{/each}
  </nav>
  <div id="examples" tabindex="-1">
    {#each groups as group (group.id)}
      <section class="category" id={group.id} aria-labelledby={`heading-${group.id}`}>
        <header class="category-heading">
          <h2 id={`heading-${group.id}`}>{group.title}</h2>
          <p>{group.description}</p>
        </header>
        {#each group.examples as example (example.anchor)}<Example {example} {dark} />{/each}
      </section>
    {/each}
  </div>
  <div class="closing">
    <p>Find the edges of what's supported.</p>
    <a href={checklistURL} target="_blank" rel="noreferrer">Read the feature checklist <span aria-hidden="true">↗</span></a>
  </div>
</main>
<footer>
  <p>sumi <span aria-hidden="true">墨</span> — diagrams laid out in pure Go.</p>
  <p>Default layout settings and sumi's own palettes. <a href="https://github.com/bencode/sumi#license">License</a></p>
</footer>
