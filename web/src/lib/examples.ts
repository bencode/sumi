import catalog from '../../../docs/examples/catalog.json'

type ExampleMetadata = {
  id: string
  title: string
  description: string
}

type GroupMetadata = {
  id: string
  title: string
  description: string
  examples: ExampleMetadata[]
}

export type Example = ExampleMetadata & {
  anchor: string
  source: string
}

export type Group = Omit<GroupMetadata, 'examples'> & { examples: Example[] }

const metadata: { groups: GroupMetadata[] } = catalog
const sources = import.meta.glob<string>('../../../docs/examples/**/*.mmd', {
  query: '?raw',
  import: 'default',
  eager: true,
})

export const groups: Group[] = metadata.groups.map(group => ({
  ...group,
  examples: group.examples.map(example => {
    const path = `../../../docs/examples/${group.id}/${example.id}.mmd`
    const source = sources[path]
    if (source === undefined) throw new Error(`Missing example source: ${path}`)
    return { ...example, anchor: `${group.id}-${example.id}`, source }
  }),
}))

export const exampleCount = groups.reduce((count, group) => count + group.examples.length, 0)
