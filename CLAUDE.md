# Regras do Projeto

## Após alterações no código

- Sempre verificar e atualizar a documentação em `docs/` para refletir as mudanças feitas no projeto.

## Após alterações em HTML, CSS, ícones ou assets

**IMPORTANTE**: Sempre que modificar arquivos HTML (`.tpl.html`), CSS, ícones ou qualquer asset visual, é **OBRIGATÓRIO** fazer o build:

```bash
npm run build:tailwind && npm run compress
```

### O que esse comando faz:
1. `npm run build:tailwind` - Compila e minifica o CSS do Tailwind
2. `npm run compress` - Comprime os arquivos CSS e JS com Brotli

### Quando executar:
- Após modificar qualquer arquivo `.tpl.html` (templates)
- Após adicionar/modificar classes CSS do Tailwind
- Após adicionar/modificar ícones (Iconify)
- Após modificar arquivos em `static/assets/`

### Por que é necessário:
O Tailwind CSS trabalha com JIT (Just-In-Time) e precisa escanear os templates para gerar apenas as classes CSS que são realmente usadas. Sem o build, as novas classes não estarão disponíveis e os ícones podem não carregar corretamente.

## Como adicionar novos ícones (Iconify)

O sistema usa um **bundle customizado** do Iconify em vez de carregar ícones dinamicamente via API. Isso garante que todos os ícones necessários estejam disponíveis offline.

### Para adicionar um novo ícone:

1. **Adicione o ícone ao bundle** em `scripts/bundle_icons.js`:
   ```javascript
   let icons = [
       'bi:people-fill',
       'mdi:web',          // Adicione o novo ícone aqui
       // ... outros ícones
   ]
   ```

2. **Regenere o bundle de ícones**:
   ```bash
   node scripts/bundle_icons.js
   ```

3. **Faça o build dos assets**:
   ```bash
   npm run build:tailwind && npm run compress
   ```

4. **Use o ícone no template**:
   ```html
   <span class="iconify inline text-xl text-accent" data-icon="mdi:web"></span>
   ```

### Conjuntos de ícones disponíveis:
- `bi:*` - Bootstrap Icons
- `mdi:*` - Material Design Icons
- `eva:*` - Eva Icons
- `ic:*` - Google Material Icons
- `fa-regular:*` - Font Awesome Regular
- `twemoji:*` - Twitter Emoji
- E muitos outros (veja https://iconify.design/)

### Importante:
- Só adicione ícones que são **realmente usados** no projeto
- O bundle customizado mantém o tamanho do JavaScript otimizado
- Se um ícone não aparecer, verifique se ele está no bundle (`scripts/bundle_icons.js`)
