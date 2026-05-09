FROM hashicorp/terraform:1.8

WORKDIR /infra/envs/dev

# Pré-instalar os providers na imagem para evitar timeout de plugin
# (o binário do provider AWS tem ~675MB e demora para iniciar via volume Windows)
COPY envs/dev/.terraform.lock.hcl .terraform.lock.hcl
COPY envs/dev/main.tf main.tf
COPY envs/dev/variables.tf variables.tf
COPY envs/dev/outputs.tf outputs.tf
COPY modules /infra/modules

RUN terraform init -backend=false -input=false

WORKDIR /infra
