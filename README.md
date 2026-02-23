# Devtools (`devv`)

Um conjunto de ferramentas avançadas de linha de comando (CLI) arquitetadas para otimizar fluxos de desenvolvimento locais utilizando ZSH, MySQL, tmux e fzf.

O antigo core baseado em `Justfile` foi substituído por uma **arquitetura de Roteador em Bash** interativa e global chamada **`devv`**.

## 🚀 Instalação (Ubuntu / WSL)

Para configurar globalmente o motor na sua máquina, instanciar o autocomplete no ZSH e baixar todas as dependências interativas, rode:

```bash
cd ~/workspace/personal
git clone git@github.com:VitorAllux/devtools.git
cd devtools

# Setup Automático Completo:
./bin/devv env:setup
```

> Após o setup, reinicie seu terminal ou rode `exec zsh`. O comando local `devv` agora estará ativo globalmente.

---

## 💻 Catálogo de Comandos

Todos os comandos possuem ZSH Autocompletion Nativo. Digite `devv <TAB>` em qualquer lugar do seu terminal para navegar pelas descrições em inglês.

### Gerenciamento de Banco de Dados (`db`)

Painéis interativos flutuantes (`fzf`) para operar sobre seus schemas locais do MySQL.

- **`devv db:create`**: Menu dinâmico para input do nome e criação de um novo schema usando codificação moderna UTF-8 (`utf8mb4_unicode_ci`).
- **`devv db:drop`**: Lista via menu dinâmico flutuante todos os BDs locais. Selecione o banco usando as setas para apagar. Inclui etapa de "Alerta Vermelho" e confirmação de segurança antes da exclusão estrutural.
- **`devv db:truncate`**: Selecione um schema no _fzf_ para limpá-lo inteiramente e sumariamente pulando restrições de chaves-estrangeiras (Foreign Keys constraint bypass).
- **`devv db:import`**: Utilitário colossal que faz download seguro e blindado de dumps SQL comprimidos (`.sql.gz`) hospedados no Google Drive através do rclone, realizando streaming em tempo-real do unzip + MySQL usando `pv` para tracking de progresso em MB/s. Ele **lista os bancos de dados interativamente via fzf** na tela para você escolher onde jogar o dump ou criar um novo on-the-fly.

### DevOps & SaaS Tmux (`tmux`, `api`, `web`)

Scripts de ciclo de vida projetados especificamente para seu ecossistema tmux local.

- **`devv tmux:up`**: Cria (ou atacha) uma sessão paralela no painel (`dev`) separando os contextos visualmente num grid vertical enxuto.
  - Sobe o `php artisan serve` no Pane API.
  - Sobe o background `php artisan horizon` no Pane Horizon.
  - Sobe o `npm run serve` no Pane Web.
- **`devv tmux:down`**: Encerra permanentemente a sessão `my-project` limpando os processos órfãos.
- **`devv api:restart`**: Utilitário agressivo de flush back-end. Ele paralisa as abas locais (Ctrl+C), roda o clear do cache corporativo, optimizadores e redis-cli, mata as filas do Queue / Horizon e reinicia tudo de novo purificado, economizando muitos gigabytes de digitação diária.
- **`devv web:restart`**: Restart exclusivo e imediato do builder do npm (frontend).

---

🛠 **Desenvolvido com Bash Router, UI em Fzf e Automapping de Zsh.**
