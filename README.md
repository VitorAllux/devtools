# devv

CLI pessoal para automacao de desenvolvimento local (tmux, MySQL, SSH e utilitarios).

Versao atual: `1.0.0`

## O que este projeto faz

- Padroniza tarefas repetitivas em comandos curtos.
- Centraliza setup de shell/autocomplete e atalhos.
- Mantem dados sensiveis fora do Git com backup criptografado.

## Instalacao rapida

```bash
git clone git@github.com:VitorAllux/devtools.git
cd devtools
./bin/devv env:setup
exec zsh
```

`bw` (Bitwarden CLI) e opcional no `env:setup`; ao final, o comando pergunta se voce quer instalar.

## Passo a passo completo de segredos (Bitwarden + age)

### 1) Primeira configuracao (maquina atual)

1. Rode `devv env:setup`.
2. Rode `bw login`.
3. Rode `export BW_SESSION="$(bw unlock --raw)"`.
4. Rode `bw sync`.
5. Rode `devv env:bootstrap` para gerar/validar chave age e backup criptografado.
6. Copie a chave privada local (`cat ~/.config/devv/keys/age.key`).
7. Crie um item no Bitwarden (Secure Note) chamado `devv-age-key` e cole a chave no campo Notes.
8. Pegue o ID do item: `bw list items --search "devv-age-key" | jq -r '.[0].id'`.
9. Salve no devv: `devv config:set DEVT_BW_AGE_KEY_ITEM "<ITEM_ID>"`.
10. Rode `devv env:bootstrap --force` para validar restauracao completa.

### 2) Recuperacao apos formatar PC

1. Clone o repositorio e rode `devv env:setup`.
2. Rode `bw login`.
3. Rode `export BW_SESSION="$(bw unlock --raw)"`.
4. Rode `bw sync`.
5. Rode `devv env:bootstrap --force`.

Resultado esperado:

- `~/.config/devv/keys/age.key` restaurada (ou existente).
- `~/.config/devv/servers.list` restaurado do `secrets/servers.list.age`.

### 3) Validacao rapida

```bash
ls -l ~/.config/devv/keys/age.key ~/.config/devv/servers.list
devv ssh:connect
```

## Onde os dados ficam

- Lista SSH local: `~/.config/devv/servers.list`
- Chave age privada local: `~/.config/devv/keys/age.key`
- Backup criptografado versionado: `secrets/servers.list.age`
- Lista de recipients publicos: `secrets/age-recipients.txt`
- Configuracao local do devv: `~/.config/devv/config.env`

O arquivo antigo `config/servers.list` foi descontinuado e substituido por `config/servers.list.example`.

## Comandos principais

### Ambiente

- `devv env:setup`
- `devv env:bootstrap`

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
- `devv ssh:list`

### WSL

- `devv wsl:list`
- `devv wsl:status`
- `devv wsl:start <distro>`
- `devv wsl:stop <distro>`
- `devv wsl:shutdown`

### Interface grafica (Control Center)

- `devv ui`

### Configuracao

- `devv config:set <KEY> <VALUE>`
- `devv config:list`

## Estrutura relevante

- `bin/devv`: roteador principal.
- `src/commands`: comandos do CLI.
- `src/lib/config.sh`: leitura/escrita em `~/.config/devv/config.env`.
- `src/lib/secrets.sh`: fluxo de segredo local + backup criptografado.
- `src/control_center/app.py`: UI desktop (fase 1) para gerir WSL/SSH/config.

## Observacoes

- Este projeto nao usa banco proprio para persistencia do devv.
- Os comandos `db:*` operam sobre bancos MySQL locais de outros projetos.
