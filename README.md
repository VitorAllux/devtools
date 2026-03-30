# devv

CLI pessoal para acelerar rotina local de desenvolvimento (tmux, MySQL, SSH e utilidades de ambiente).

Versao atual: `1.0.0`

## Objetivo

- Padronizar tarefas repetitivas em comandos curtos.
- Centralizar setup de shell/autocomplete e atalhos.
- Manter dados sensiveis fora do Git e recuperaveis apos formatacao.

## Instalacao

```bash
git clone git@github.com:VitorAllux/devtools.git
cd devtools
./bin/devv env:setup
exec zsh
```

## Comandos principais

### Ambiente

- `devv env:setup`: instala dependencias base e integra shell/autocomplete.
- `devv env:bootstrap`: roda setup + restaura arquivos sensiveis locais.

### Banco (MySQL local)

- `devv db:create`
- `devv db:drop`
- `devv db:truncate`
- `devv db:import`
- `devv db:clean`
- `devv db:ui`

### Tmux

- `devv tmux:up`
- `devv tmux:down`
- `devv tmux:session`
- `devv tmux:window`
- `devv api:restart`
- `devv web:restart`

### SSH

- `devv ssh:connect`
- `devv ssh:add`
- `devv ssh:remove`

### Configuracao

- `devv config:set <KEY> <VALUE>`
- `devv config:list`

## Seguranca e segredos

Arquivos sensiveis nao ficam mais versionados em texto puro.

- Lista SSH local: `~/.config/devv/servers.list`
- Chave privada age local: `~/.config/devv/keys/age.key`
- Backup criptografado no repo: `secrets/servers.list.age`
- Recipients publicos: `secrets/age-recipients.txt`

O arquivo antigo `config/servers.list` foi descontinuado (mantido apenas `config/servers.list.example`).

## Fluxo de recuperacao (formatou o PC)

1. Clone o repositorio.
2. Rode `devv env:bootstrap`.
3. Se houver backup criptografado, o comando tenta restaurar a chave via Bitwarden CLI (`bw`).
4. Se a chave for restaurada, o `servers.list` local e reidratado automaticamente.

Para informar o item do Bitwarden que guarda a chave age:

```bash
devv config:set DEVT_BW_AGE_KEY_ITEM "<item-id-ou-nome>"
```

## Estrutura relevante

- `bin/devv`: roteador principal de comandos.
- `src/commands`: comandos do CLI.
- `src/lib/ui.sh`: helpers visuais e utilitarios.
- `src/lib/config.sh`: configuracoes em `~/.config/devv/config.env`.
- `src/lib/secrets.sh`: gerenciamento de segredo local + backup criptografado.

## Observacoes

- Este projeto nao usa banco proprio para persistir configuracoes do devv.
- Os comandos `db:*` operam sobre bancos MySQL locais dos seus projetos.
