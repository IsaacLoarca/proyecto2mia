import { useState, useRef } from 'react';
import Editor from '@monaco-editor/react';
import type { editor } from 'monaco-editor';

interface CodeEditorProps {
  initialValue?: string;
  onValueChange?: (value: string) => void;
}

const CodeEditor = ({ initialValue = '', onValueChange }: CodeEditorProps) => {
  const [value, setValue] = useState(initialValue);
  const editorRef = useRef<editor.IStandaloneCodeEditor | null>(null);

  const handleEditorDidMount = (editor: editor.IStandaloneCodeEditor) => {
    editorRef.current = editor;
    editor.getModel()?.updateOptions({ 
      tabSize: 2,
      insertSpaces: true 
    });
  };

  const handleEditorChange = (value: string | undefined) => {
    const newValue = value || '';
    setValue(newValue);
    onValueChange?.(newValue);
  };

  return (
    <div className="editor-container">
      <Editor
        height="100%"
        language="plaintext"
        theme="vs-dark"  // Usar tema por defecto
        value={value}
        onMount={handleEditorDidMount}
        onChange={handleEditorChange}
        options={{
          minimap: { enabled: false },
          fontSize: 14,
          lineNumbers: 'on',
          roundedSelection: false,
          scrollBeyondLastLine: false,
          automaticLayout: true,
          wordWrap: 'on',
          wrappingIndent: 'indent',
          lineHeight: 20,
          folding: false,
          glyphMargin: false,
          renderWhitespace: 'selection',
          padding: { top: 10, bottom: 10 },
        }}
      />
    </div>
  );
};

export default CodeEditor;