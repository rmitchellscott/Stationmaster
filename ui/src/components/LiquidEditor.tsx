import React, { useRef, useEffect } from "react";
import Editor, { BeforeMount, OnMount } from "@monaco-editor/react";
import { useTheme } from "next-themes";

interface LiquidEditorProps {
  value: string;
  onChange: (value: string) => void;
  placeholder?: string;
  height?: string | number;
  readOnly?: boolean;
  language?: string;
}

export function LiquidEditor({ 
  value, 
  onChange, 
  placeholder = "Enter Liquid template...",
  height = "300px",
  readOnly = false,
  language = "liquid"
}: LiquidEditorProps) {
  const { theme } = useTheme();
  const editorRef = useRef<any>(null);

  // Register Liquid language with Monaco
  const handleEditorWillMount: BeforeMount = (monaco) => {
    // Register Liquid language if not already registered
    const languages = monaco.languages.getLanguages();
    const liquidLangExists = languages.some(lang => lang.id === 'liquid');
    
    if (!liquidLangExists) {
      monaco.languages.register({ id: 'liquid' });

      // Define Liquid tokenizer
      monaco.languages.setMonarchTokenizer('liquid', {
        tokenizer: {
          root: [
            // Liquid output tags {{ }}
            [/\{\{/, { token: 'delimiter.liquid', bracket: '@open', next: '@liquidOutput' }],
            
            // Liquid logic tags {% %}
            [/\{%/, { token: 'delimiter.liquid', bracket: '@open', next: '@liquidLogic' }],
            
            // HTML tags
            [/<\/?[a-zA-Z][\w-]*/, 'tag'],
            [/<\/?[a-zA-Z][\w-]*\s*\/?>/, 'tag'],
            
            // HTML attributes
            [/\s+[a-zA-Z-]+(?==)/, 'attribute.name'],
            [/="[^"]*"/, 'attribute.value'],
            [/='[^']*'/, 'attribute.value'],
            
            // CSS classes (for TRMNL framework)
            [/class\s*=\s*"[^"]*"/, 'string.css-class'],
            [/class\s*=\s*'[^']*'/, 'string.css-class'],
            
            // Comments
            [/\{%\s*comment\s*%\}.*?\{%\s*endcomment\s*%\}/s, 'comment'],
            [/<!--.*?-->/s, 'comment'],
            
            // Strings
            [/"([^"\\]|\\.)*"/, 'string'],
            [/'([^'\\]|\\.)*'/, 'string'],
            
            // Numbers
            [/\d+\.?\d*/, 'number'],
            
            // Other HTML content
            [/[^{<]+/, 'text'],
          ],
          
          liquidOutput: [
            [/\}\}/, { token: 'delimiter.liquid', bracket: '@close', next: '@pop' }],
            [/\|/, 'operator.liquid'], // Filters
            [/[a-zA-Z_]\w*/, 'variable.liquid'],
            [/\./, 'operator.liquid'],
            [/\d+\.?\d*/, 'number'],
            [/"([^"\\]|\\.)*"/, 'string'],
            [/'([^'\\]|\\.)*'/, 'string'],
            [/\s+/, 'white'],
            [/:/, 'operator.liquid'],
            [/,/, 'delimiter.liquid'],
            [/[^}]+/, 'text'],
          ],
          
          liquidLogic: [
            [/%\}/, { token: 'delimiter.liquid', bracket: '@close', next: '@pop' }],
            [/\b(if|elsif|else|endif|unless|endunless|case|when|endcase|for|endfor|while|endwhile|break|continue|assign|capture|endcapture|include|render|layout|extends|block|endblock)\b/, 'keyword.liquid'],
            [/\b(in|and|or|not|contains|empty|nil|blank)\b/, 'operator.liquid'],
            [/[a-zA-Z_]\w*/, 'variable.liquid'],
            [/\./, 'operator.liquid'],
            [/\d+\.?\d*/, 'number'],
            [/"([^"\\]|\\.)*"/, 'string'],
            [/'([^'\\]|\\.)*'/, 'string'],
            [/\s+/, 'white'],
            [/[=<>!]=?/, 'operator.liquid'],
            [/[+\-*/]/, 'operator.liquid'],
            [/:/, 'operator.liquid'],
            [/,/, 'delimiter.liquid'],
            [/\[|\]/, 'bracket.liquid'],
            [/\(|\)/, 'bracket.liquid'],
            [/[^%]+/, 'text'],
          ]
        }
      });

      // Configure language features
      monaco.languages.setLanguageConfiguration('liquid', {
        brackets: [
          ['{{', '}}'],
          ['{%', '%}'],
          ['<', '>'],
          ['[', ']'],
          ['(', ')']
        ],
        autoClosingPairs: [
          { open: '{{', close: '}}' },
          { open: '{%', close: '%}' },
          { open: '<', close: '>' },
          { open: '[', close: ']' },
          { open: '(', close: ')' },
          { open: '"', close: '"' },
          { open: "'", close: "'" }
        ],
        surroundingPairs: [
          { open: '{{', close: '}}' },
          { open: '{%', close: '%}' },
          { open: '<', close: '>' },
          { open: '[', close: ']' },
          { open: '(', close: ')' },
          { open: '"', close: '"' },
          { open: "'", close: "'" }
        ]
      });

      // Define color theme for Liquid
      monaco.editor.defineTheme('liquid-dark', {
        base: 'vs-dark',
        inherit: true,
        rules: [
          { token: 'delimiter.liquid', foreground: 'ff6188' },
          { token: 'keyword.liquid', foreground: 'ff6188', fontStyle: 'bold' },
          { token: 'operator.liquid', foreground: 'a9dc76' },
          { token: 'variable.liquid', foreground: 'ffd866' },
          { token: 'string.css-class', foreground: 'ab9df2', fontStyle: 'italic' },
          { token: 'tag', foreground: '78dce8' },
          { token: 'attribute.name', foreground: 'a9dc76' },
          { token: 'attribute.value', foreground: 'ffd866' },
        ],
        colors: {}
      });

      monaco.editor.defineTheme('liquid-light', {
        base: 'vs',
        inherit: true,
        rules: [
          { token: 'delimiter.liquid', foreground: 'c41e3a' },
          { token: 'keyword.liquid', foreground: 'c41e3a', fontStyle: 'bold' },
          { token: 'operator.liquid', foreground: '22863a' },
          { token: 'variable.liquid', foreground: 'b08800' },
          { token: 'string.css-class', foreground: '6f42c1', fontStyle: 'italic' },
          { token: 'tag', foreground: '005cc5' },
          { token: 'attribute.name', foreground: '22863a' },
          { token: 'attribute.value', foreground: 'b08800' },
        ],
        colors: {}
      });
    }

    // Add completion provider for TRMNL framework classes and Liquid variables
    monaco.languages.registerCompletionItemProvider('liquid', {
      provideCompletionItems: (model, position) => {
        const suggestions = [];

        // TRMNL framework CSS classes
        const trmnlClasses = [
          'screen', 'screen--portrait', 'screen--no-bleed', 'screen--dark-mode',
          'view', 'view--full', 'view--half_vertical', 'view--half_horizontal', 'view--quadrant',
          'layout', 'layout--vertical', 'layout--horizontal', 'layout--compact',
          'columns', 'column', 'column--auto', 'column--grow',
          'title', 'title--small', 'title--large',
          'label', 'label--small', 'label--muted',
          'text', 'text--small', 'text--muted', 'text--center', 'text--right',
          'gap--small', 'gap--medium', 'gap--large',
          'card', 'list', 'item', 'badge', 'progress',
          'flex', 'flex--column', 'flex--wrap', 'flex--center',
          'grid', 'grid--2', 'grid--3', 'grid--4',
          'p--small', 'p--medium', 'p--large',
          'm--small', 'm--medium', 'm--large'
        ];

        trmnlClasses.forEach(className => {
          suggestions.push({
            label: className,
            kind: monaco.languages.CompletionItemKind.Class,
            insertText: className,
            documentation: `TRMNL framework CSS class: ${className}`
          });
        });

        // Liquid variables
        const liquidVariables = [
          'data', 'trmnl.user.first_name', 'trmnl.user.email',
          'trmnl.device.name', 'trmnl.device.width', 'trmnl.device.height',
          'trmnl.timestamp', 'layout.type', 'layout.width', 'layout.height',
          'layout.is_split', 'instance_id'
        ];

        liquidVariables.forEach(variable => {
          suggestions.push({
            label: variable,
            kind: monaco.languages.CompletionItemKind.Variable,
            insertText: variable,
            documentation: `Available Liquid variable: {{ ${variable} }}`
          });
        });

        // Liquid filters
        const liquidFilters = [
          'truncate', 'escape_html', 'safe', 'format_date', 'format_time',
          'upcase', 'downcase', 'capitalize', 'strip', 'lstrip', 'rstrip',
          'replace', 'remove', 'split', 'join', 'reverse', 'sort',
          'first', 'last', 'size', 'default'
        ];

        liquidFilters.forEach(filter => {
          suggestions.push({
            label: filter,
            kind: monaco.languages.CompletionItemKind.Function,
            insertText: filter,
            documentation: `Liquid filter: | ${filter}`
          });
        });

        // Liquid tags
        const liquidTags = [
          'if', 'elsif', 'else', 'endif', 'unless', 'endunless',
          'case', 'when', 'endcase', 'for', 'in', 'endfor',
          'assign', 'capture', 'endcapture', 'comment', 'endcomment'
        ];

        liquidTags.forEach(tag => {
          suggestions.push({
            label: tag,
            kind: monaco.languages.CompletionItemKind.Keyword,
            insertText: tag,
            documentation: `Liquid tag: {% ${tag} %}`
          });
        });

        return { suggestions };
      }
    });
  };

  const handleEditorDidMount: OnMount = (editor, monaco) => {
    editorRef.current = editor;

    // Set theme based on current theme
    const currentTheme = theme === 'dark' ? 'liquid-dark' : 'liquid-light';
    monaco.editor.setTheme(currentTheme);

    // Configure editor options
    editor.updateOptions({
      fontSize: 13,
      lineHeight: 20,
      fontFamily: 'Monaco, Menlo, "Ubuntu Mono", monospace',
      minimap: { enabled: false },
      scrollBeyondLastLine: false,
      automaticLayout: true,
      wordWrap: 'on',
      bracketPairColorization: { enabled: true },
      guides: {
        bracketPairs: true,
        indentation: true
      }
    });
  };

  // Set initial value and handle value changes
  useEffect(() => {
    if (editorRef.current) {
      const currentValue = editorRef.current.getValue();
      const targetValue = value || (placeholder ? `<!-- ${placeholder} -->\n` : '');
      
      if (currentValue !== targetValue) {
        editorRef.current.setValue(targetValue);
        
        // Set cursor position for placeholder
        if (!value && placeholder) {
          const monaco = (window as any).monaco;
          if (monaco) {
            editorRef.current.setSelection(new monaco.Selection(2, 1, 2, 1));
          }
        }
      }
    }
  }, [value, placeholder]);

  // Update theme when it changes
  useEffect(() => {
    if (editorRef.current) {
      const monaco = (window as any).monaco;
      if (monaco) {
        const currentTheme = theme === 'dark' ? 'liquid-dark' : 'liquid-light';
        monaco.editor.setTheme(currentTheme);
      }
    }
  }, [theme]);

  return (
    <div className="border rounded-md overflow-hidden">
      <Editor
        height={height}
        language="liquid"
        value={value}
        onChange={(val) => onChange(val || '')}
        beforeMount={handleEditorWillMount}
        onMount={handleEditorDidMount}
        options={{
          readOnly,
          theme: theme === 'dark' ? 'liquid-dark' : 'liquid-light',
        }}
      />
    </div>
  );
}