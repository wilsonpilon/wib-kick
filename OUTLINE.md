# Sidekick clone — andamento do projeto

Resumo do que já foi feito e onde paramos, para retomar amanhã.

## Feito

### Menu S&K
- Item **E&xit** já existia no menu S&K.
- Corrigido o **Alt+K não abrir o menu** no Windows. Duas causas:
  1. O driver nativo do Windows Console no tcell v2.8.1 descartava eventos
     Alt+letra. Corrigido atualizando `github.com/gdamore/tcell/v2` para
     v2.13.10 (Alt+letra é essencial, não pode ser substituído por F10).
  2. Bug de ordem de desenho: `Desktop.Draw` sobrescrevia a área do dropdown
     do menu. Corrigido separando `MenuBar.Draw` em `DrawBar`/`DrawOverlay`,
     com `DrawOverlay` sempre desenhado por último em `App.Draw`.
- Adicionada **borda simples estilo MS-DOS** (`┌─┐│└┘`) ao redor dos
  dropdowns/submenus (`drawBorder` em `internal/tv/menubar.go`).
- `MenuBar` reescrito para suportar **múltiplos títulos** no topo (antes só
  existia "S&K"): `menuTitle`, `titles []*menuTitle`, `OpenByHotkey`,
  `switchTitle` (setas esquerda/direita entre títulos), `ToggleTitle`, etc.
- Campo `shortcut` adicionado a `item` (`action(label, id, shortcut...)`),
  exibido alinhado à direita nos itens do menu (ex: "Alt+N", "F3").

### Menu &File (construído, mas guardado / não usado ainda)
- Percebemos que o menu do Sidekick é **dinâmico**: muda conforme o
  acessório ativo (S&K, Time Planner, etc.), não é uma barra fixa com vários
  títulos permanentes.
- O menu **&File** (New/Open/Defaults/Setup reconcile/Reconcile day/Tools
  [Copy/Rename/Delete]/Protect/Unprotect/Print/Printer settings/Close) foi
  construído e depois **guardado** como função não usada
  `fileMenuTitle()` em `internal/tv/app.go` — pertence ao futuro acessório
  **&Time Planner**, não deve aparecer agora. Hoje só "S&K" é exibido.

### Janela flutuante &Notepad (Turbo Vision / Borland style)
Implementado do zero (arquivos novos):
- `internal/theme/theme.go`: cores `WindowBg` (azul), `WindowFg` (amarelo),
  `WindowBorder` (branco), `WindowAccent` (verde), `ShadowBg` (preto).
- `internal/tv/window.go` (novo): chrome genérico de janela flutuante —
  borda dupla (`╔═╗║╚╝`), sombra estilo Borland, barra de título com
  `[■]` (fechar), número da janela, `[↑]`/`[↕]` (maximizar/restaurar),
  scrollbars proporcionais nas bordas direita/inferior, hit-test e
  arrastar/redimensionar/maximizar (`Window`, `WindowContent` interface).
- `internal/tv/editor.go` (novo): conteúdo de texto do Notepad — digitação,
  Enter/Backspace/Delete, setas/Home/End/Tab, rolagem acompanhando o
  cursor, régua de coluna (`▶` na coluna visível 0, `·` de preenchimento,
  números a cada 10 colunas, `▼` a cada tab stop).
- `internal/tv/app.go`: `App` agora guarda `windows []*Window`,
  cascata de novas janelas, foco/z-order (`raiseWindow`/`closeWindow`/
  `focusedWindow`), roteamento de teclado/mouse (drag/resize/close/zoom/
  clique no conteúdo), e posicionamento do cursor do terminal.
- `go build ./...` e `go vet ./...` passam limpos.

## Onde paramos (pendente para amanhã)

**Bug/ajuste visual reportado pelo usuário:** a sombra da janela ficou com
as proporções trocadas em relação ao pedido original ("1 caractere de
largura, 2 de altura"). Revisei o código de `DrawShadow`
(`internal/tv/window.go`) e ele **implementa exatamente** o que foi pedido
literalmente (tira direita = 1 coluna de largura, tira inferior = 2 linhas
de altura) — não há bug de troca no código.

Hipótese: como os caracteres do terminal são mais altos que largos (~1:2),
essa proporção literal (1 largura / 2 altura) visualmente fica "invertida"
em relação à sombra clássica dos programas Borland/Turbo Vision, que
normalmente é **2 colunas à direita e 1 linha embaixo** (para que as duas
tiras pareçam ter a mesma espessura visual).

**Pergunta em aberto, aguardando resposta do usuário:** trocar a sombra
para o padrão clássico — tira direita com **2 caracteres de largura**, tira
inferior com **1 caractere de altura**? Se confirmado, o ajuste é só nos
limites dos dois loops em `Window.DrawShadow` (`internal/tv/window.go`).

## Próximos passos sugeridos
1. Resolver a proporção da sombra (aguardando confirmação acima).
2. Testar manualmente no console Windows (não consigo testar interativamente
   por aqui): abrir Notepad, digitar, rolar, arrastar título, redimensionar
   pelo canto, fechar `[■]`, maximizar/restaurar `[↑]`/`[↕]`, abrir várias
   notas (cascata) e trocar foco clicando entre elas.
3. Quando o Time Planner for implementado, ligar `fileMenuTitle()` à troca
   dinâmica de menu por acessório ativo.
