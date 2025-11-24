import { useEditor, EditorContent } from '@tiptap/react'
import StarterKit from '@tiptap/starter-kit'

export function Editor({ content }) {
    const editor = useEditor({
        extensions: [
            StarterKit,
        ],
        content: content || `
      <h2>Welcome back</h2>
      <p>Select an activity from the timeline to view details.</p>
      <p>This is a <strong>rich text editor</strong> where your memories will be summarized.</p>
      <ul>
        <li>Auto-generated summaries</li>
        <li>Editable content</li>
        <li>Block-based structure</li>
      </ul>
    `,
        editorProps: {
            attributes: {
                class: 'prose prose-sm sm:prose lg:prose-lg xl:prose-2xl mx-auto focus:outline-none dark:prose-invert',
            },
        },
    })

    return <EditorContent editor={editor} />
}
