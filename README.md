# SDK Go da Assinafy

*Português · [Read in English](README.en.md)*

[![CI](https://github.com/assinafy/golang-sdk/actions/workflows/ci.yml/badge.svg)](https://github.com/assinafy/golang-sdk/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/assinafy/golang-sdk.svg)](https://pkg.go.dev/github.com/assinafy/golang-sdk)

Cliente Go para a [API Assinafy v1](https://api.assinafy.com.br/v1/docs) — plataforma
brasileira de assinatura eletrônica de documentos. Cobre as 93 operações da descrição
OpenAPI publicada, com requisições e respostas tipadas e um contrato de erro único.

## Conteúdo

1. [Requisitos](#requisitos)
2. [Instalação](#instalação)
3. [Início rápido](#início-rápido)
4. [Configurando o cliente](#configurando-o-cliente)
5. [Como toda chamada se comporta](#como-toda-chamada-se-comporta)
6. [O ciclo de vida da assinatura](#o-ciclo-de-vida-da-assinatura)
   - [1. Envie o documento](#1-envie-o-documento)
   - [2. Espere os metadados das páginas](#2-espere-os-metadados-das-páginas)
   - [3. Crie os signatários](#3-crie-os-signatários)
   - [4. Estime o custo](#4-estime-o-custo)
   - [5. Solicite as assinaturas](#5-solicite-as-assinaturas)
   - [6. Acompanhe a solicitação](#6-acompanhe-a-solicitação)
   - [7. Conduza a sessão do signatário](#7-conduza-a-sessão-do-signatário)
   - [8. Baixe o documento certificado](#8-baixe-o-documento-certificado)
   - [O fluxo inteiro em uma chamada](#o-fluxo-inteiro-em-uma-chamada)
7. [Verificação e notificação do signatário](#verificação-e-notificação-do-signatário)
   - [Assinando com certificado ICP-Brasil](#assinando-com-certificado-icp-brasil)
8. [Conectando a conta de outras pessoas com OAuth](#conectando-a-conta-de-outras-pessoas-com-oauth)
9. [Criando documentos a partir de modelos](#criando-documentos-a-partir-de-modelos)
10. [Organizando documentos com etiquetas e campos](#organizando-documentos-com-etiquetas-e-campos)
11. [Recebendo webhooks](#recebendo-webhooks)
12. [Contas, usuários e estatísticas](#contas-usuários-e-estatísticas)
13. [Erros e novas tentativas](#erros-e-novas-tentativas)
14. [Cuidando das credenciais](#cuidando-das-credenciais)
15. [Cobertura da API](#cobertura-da-api)
16. [Ambientes](#ambientes)
17. [Desenvolvimento](#desenvolvimento)
18. [Licença](#licença)

## Requisitos

- Go 1.26 ou superior. O mínimo do módulo é a diretiva `go 1.26` no `go.mod`.
- Nenhuma dependência de terceiros: o SDK usa apenas a biblioteca padrão.
- TLS 1.2 ou superior. Os clientes HTTP padrão do SDK recusam TLS 1.0 e 1.1; um cliente que você fornecer mantém as próprias configurações de TLS.

Go não designa releases como LTS — o time do Go dá suporte às duas versões maiores
mais recentes. A CI compila e testa nas duas (Go 1.26.x e 1.27.x) e roda as
verificações de formatação, `vet`, vulnerabilidade e lint no Go 1.27.x.

## Instalação

```bash
go get github.com/assinafy/golang-sdk
```

## Início rápido

Este programa completo faz uma requisição somente-leitura ao sandbox. Mantenha
credenciais em variáveis de ambiente — nunca no código-fonte.

```go
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	assinafy "github.com/assinafy/golang-sdk"
)

func main() {
	client, err := assinafy.NewClient(assinafy.ClientOptions{
		APIKey:    os.Getenv("ASSINAFY_API_KEY"),
		AccountID: os.Getenv("ASSINAFY_ACCOUNT_ID"),
		BaseURL:   assinafy.SandboxBaseURL,
	})
	if err != nil {
		log.Fatal(err)
	}

	contas, err := client.Accounts.List(context.Background())
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("contas: %d\n", len(contas))
}
```

## Configurando o cliente

`assinafy.NewClient` valida suas opções e constrói um cliente seguro para uso
concorrente, sem fazer nenhuma requisição de rede. Um cliente pode ser
compartilhado entre goroutines por toda a vida do processo.

| Opção | Tipo | Padrão | Descrição |
| --- | --- | --- | --- |
| `APIKey` | `string` | vazio | Credencial permanente, enviada como `X-Api-Key`. |
| `Token` | `string` | vazio | Token de acesso, enviado como `Authorization: Bearer`; usado quando `APIKey` está vazio. |
| `TokenSource` | `assinafy.TokenSource` | nulo | Resolve um token a cada requisição, usado quando `APIKey` e `Token` estão vazios. É assim que uma conexão OAuth se autentica; veja [Conectando a conta de outras pessoas com OAuth](#conectando-a-conta-de-outras-pessoas-com-oauth). |
| `AccountID` | `string` | vazio | ID de conta padrão para recursos com escopo de conta. Um argumento não vazio no método sobrescreve. |
| `BaseURL` | `string` | `https://api.assinafy.com.br/v1` | Raiz da API. Use `assinafy.SandboxBaseURL` para o sandbox. |
| `Timeout` | `time.Duration` | `30s` | Timeout HTTP por requisição. |

**Escolhendo a credencial.** Uma chave de API é a escolha certa para automatizar a
**sua própria** conta: é permanente e tem escopo do usuário que a criou. Um token
bearer vem de `client.Authentication.Login` e expira. Um `TokenSource` é para uma
aplicação que age na conta **de outra pessoa**, com a permissão dela. Quando mais de
uma está configurada, `APIKey` vence, depois `Token`, depois `TokenSource`. As
credenciais são opcionais na construção, porque login, documento público,
verificação e as operações por código de acesso do signatário não usam nenhuma.

**Seleção de conta.** Métodos com escopo de conta recebem um argumento `accountID`.
Passar `""` cai no `ClientOptions.AccountID`; se ambos estiverem vazios, o SDK
rejeita a chamada localmente em vez de enviar um caminho malformado.

**Recursos.** O cliente agrupa a API por área:

```go
client.Accounts        client.Documents   client.Templates  client.Tags
client.Signers         client.Assignments client.Fields     client.Webhooks
client.SignerDocuments client.Users       client.Authentication
client.PublicDocuments
```

O fluxo OAuth fica no pacote `oauth`, separado, porque uma integração o chama antes
de ter um cliente.

## Como toda chamada se comporta

As mesmas poucas regras valem para todos os métodos, então vale lê-las uma vez.

- **Contexto primeiro.** Todo método recebe um `context.Context` e respeita
  cancelamento e prazos.
- **Desembrulho do envelope.** A API embrulha as respostas JSON como
  `{status, message, data}`. Os métodos devolvem o `data` já decodificado em um
  modelo; você nunca lida com o envelope. As quatro operações OAuth são a exceção:
  respondem com JSON OAuth puro, sem envelope.
- **Paginação.** Endpoints de listagem que devolvem cabeçalhos `X-Pagination-*`
  entregam um `models.PaginatedResult[T]` com `Data` e uma struct `Pagination`
  (`CurrentPage`, `TotalCount`, `PageCount`, `PerPage`). Endpoints sem esses
  cabeçalhos devolvem uma fatia simples. `models.ListParams.SetDefaults` normaliza
  `Page` para 1 e limita `PerPage` ao máximo da API, 100.
- **Conteúdo binário.** Artefatos, páginas, miniaturas, logotipo e imagens de
  assinatura voltam como `[]byte` cru. Gravá-los com nome e permissão adequados é
  responsabilidade de quem chama.
- **Campos opcionais.** Os modelos de requisição usam ponteiros para os campos que a
  API trata como opcionais; `nil` omite o campo e mantém o valor do servidor. Campos
  com contrato de três estados (a cor de uma etiqueta, o regex de um campo) expõem um
  booleano `Clear…` que envia JSON `null`.
- **Erros.** Toda falha é tipada. Veja [Erros e novas tentativas](#erros-e-novas-tentativas).

## O ciclo de vida da assinatura

Uma solicitação de assinatura virtual — o caso comum — leva um documento por oito
etapas. Cada etapa abaixo é uma chamada do SDK, nesta ordem.

### 1. Envie o documento

```go
pdf, err := os.ReadFile("contrato.pdf")
if err != nil {
	log.Fatal(err)
}

document, err := client.Documents.Upload(ctx, "", pdf, "contrato.pdf", nil)
```

O upload é `multipart/form-data`. Apenas PDF, no máximo 25 MB e 2.000 páginas — o
SDK confere o cabeçalho PDF e o limite de bytes antes de enviar; a contagem de
páginas é validada no servidor. O `accountID` vazio usa `ClientOptions.AccountID`.

O processamento continua de forma assíncrona depois da resposta, então o documento
volta com status `uploaded` ou `metadata_processing`.

### 2. Espere os metadados das páginas

Só há `Pages` para posicionar campos depois que o processamento termina.

```go
func waitForMetadata(ctx context.Context, client *assinafy.Client, documentID string) (*models.Document, error) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		document, err := client.Documents.Get(ctx, documentID)
		if err != nil {
			return nil, err
		}
		switch document.Status {
		case models.StatusMetadataReady, models.StatusReady:
			return document, nil
		case models.StatusFailed:
			return nil, fmt.Errorf("processamento do documento falhou")
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
		}
	}
}
```

Uma assinatura virtual — sem campos — não precisa das páginas e pode pular esta
etapa. Se a conta tiver webhooks, `document_metadata_ready` evita a sondagem.

### 3. Crie os signatários

Um signatário pertence à conta e é reutilizável entre documentos. O e-mail é único
por conta: reaproveite quem já existe em vez de recriar.

```go
signer, err := client.Signers.Create(ctx, "", &models.CreateSignerRequest{
	FullName: "Maria Silva",
	Email:    &email,
})
```

`client.Signers.List`, `Get`, `Update` e `Delete` completam o cadastro.
`Update` é também onde se grava o `GovernmentID` (CPF ou CNPJ), obrigatório para
quem vai assinar com certificado digital.

### 4. Estime o custo

Documentos e créditos são debitados na criação do assignment. Estime antes.

```go
estimate, err := client.Assignments.EstimateCostWithRequest(ctx, document.ID,
	&models.EstimateAssignmentCostRequest{
		Method: models.MethodVirtual,
		Signers: []models.EstimateAssignmentCostSigner{{
			VerificationMethod:  models.VerificationMethodEmail,
			NotificationMethods: []string{models.NotificationMethodEmail},
		}},
	})

if !estimate.HasSufficientResources {
	log.Fatalf("saldo insuficiente: %v", estimate.BlockingReason)
}
```

`Breakdown` detalha cada linha cobrada, `DocumentBalance` e `CreditBalance` trazem
os saldos, e `BlockingReason` diz o que impede a operação quando há bloqueio
(`PendingPayment`, `InsufficientDocuments`, `InsufficientCredits`).

### 5. Solicite as assinaturas

```go
assignment, err := client.Assignments.Create(ctx, document.ID, &models.CreateAssignmentRequest{
	Method: models.MethodVirtual,
	Signers: []models.SignerReference{{
		ID:                  signer.ID,
		VerificationMethod:  models.VerificationMethodEmail,
		NotificationMethods: []string{models.NotificationMethodEmail},
	}},
})
```

Os dois campos de método podem ser omitidos; veja
[Verificação e notificação do signatário](#verificação-e-notificação-do-signatário)
para o que cada um faz e custa.

Escolha o `Method` pelo que o signatário precisa preencher:

- `models.MethodVirtual` — o signatário aceita o documento como está. Sem campos.
- `models.MethodCollect` — o signatário preenche campos que você posiciona em
  páginas específicas, via `CreateAssignmentRequest.Entries`, cada campo com um
  `models.DisplaySettings` no sistema de coordenadas de 150 DPI da API.

Use `SignerReference.Step` para assinatura sequencial. Signatários no mesmo passo são
notificados juntos; o passo seguinte só é notificado depois que o anterior termina.
Se um signatário tiver passo, todos precisam ter, numerados de 1 em diante sem
buracos.

Criar o assignment pode disparar notificações na hora. O `SigningURLs` da resposta
traz uma URL por signatário no fluxo virtual.

### 6. Acompanhe a solicitação

| Pergunta | Chamada |
| --- | --- |
| O que aconteceu com este documento? | `client.Documents.Activities(ctx, documentID)` |
| A mensagem de WhatsApp saiu, e o que dizia? | `client.Assignments.ListWhatsAppNotifications(ctx, documentID, assignmentID)` |
| Dá para lembrar um signatário de novo? | `client.Assignments.EstimateResendCost(…)` e depois `client.Assignments.ResendNotification(…)` |
| O prazo venceu — dá para estender? | `client.Assignments.ResetExpirationWithRequest(…)` |
| O que está pendente na conta inteira? | `client.Assignments.List(ctx, params)` |

### 7. Conduza a sessão do signatário

A maioria das integrações manda o signatário para a `SigningURL` e deixa o fluxo do
navegador cuidar do resto. Quando a sua aplicação hospeda a experiência do
signatário, as mesmas etapas existem com o código de acesso dele, que o SDK envia
como o parâmetro de consulta `signer-access-code`:

```go
document, err := client.Assignments.GetSigningInfo(ctx, signerAccessCode)          // GET /sign
signer, err := client.Signers.GetSelf(ctx, signerAccessCode)                       // quem está assinando
err = client.Signers.VerifyEmail(ctx, signerAccessCode, otp)                       // código de uso único
_, err = client.Signers.ConfirmDataAndGet(ctx, documentID, signerAccessCode, body) // confirma os dados
err = client.Signers.AcceptTermsOnly(ctx, signerAccessCode)                        // aceita os termos
err = client.Signers.UploadSignatureWithReuse(ctx, signerAccessCode, "signature", true, pngBytes)
err = client.Assignments.Sign(ctx, documentID, assignmentID, signerAccessCode, items)
```

Um signatário com vários documentos pendentes pode agir em lote por
`client.SignerDocuments.SignMultiple` e `DeclineMultiple`, e listar ou buscar os
próprios documentos com `client.SignerDocuments.List` e `SearchAll`. Recusar um
assignment isolado é `client.Assignments.Decline`.

Em assignments virtuais o signatário precisa confirmar os dados antes de assinar,
então chame `ConfirmDataAndGet` antes de `Sign`.

### 8. Baixe o documento certificado

```go
func downloadWhenCertified(ctx context.Context, client *assinafy.Client, documentID, destino string) error {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for {
		document, err := client.Documents.Get(ctx, documentID)
		if err != nil {
			return err
		}
		switch document.Status {
		case models.StatusCertificated:
			pdf, err := client.Documents.Download(ctx, documentID, "certificated")
			if err != nil {
				return err
			}
			return os.WriteFile(destino, pdf, 0o600)
		case models.StatusExpired, models.StatusRejectedBySigner,
			models.StatusRejectedByUser, models.StatusFailed:
			return fmt.Errorf("assinatura encerrada com status %q", document.Status)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}
```

Cinco artefatos ficam disponíveis:

| Artefato | Conteúdo |
| --- | --- |
| `original` | O PDF exatamente como enviado. |
| `certificated` | O PDF assinado com a página de certificação da Assinafy. |
| `certificate-page` | Apenas a página de certificação. |
| `pades` | PDF com assinatura PAdES; existe só em documentos que tiveram signatário por certificado digital. |
| `bundle` | Um zip com os demais artefatos disponíveis. |

`client.Documents.Thumbnail` e `DownloadPage` devolvem imagens renderizadas, e
`client.Documents.Verify` confere um hash de assinatura sem nenhuma credencial.

### O fluxo inteiro em uma chamada

`Client.UploadAndRequestSignatures` executa as etapas 1, 3 e 5 em sequência:

```go
result, err := client.UploadAndRequestSignatures(ctx, pdfBytes, "contrato.pdf",
	[]models.UploadAndRequestSignaturesSigner{{Name: signerName, Email: signerEmail}},
	"Por favor, revise e assine", nil, "")
```

Valida cada nome, e-mail, número de WhatsApp e prazo antes da primeira requisição.
**Não** é transacional: quando uma chamada posterior falha, o resultado parcial
devolvido ainda traz o `Document` enviado e os `SignerIDs` criados, para que quem
chamou possa limpar.

## Verificação e notificação do signatário

Cada signatário de um assignment tem um **método de verificação** — como prova quem
é antes de assinar — e um **método de notificação** — como é avisado de que uma
assinatura foi pedida. Os dois são **acoplados**: envie um, os dois ou nenhum, e o
lado que faltar é inferido. Sem nenhum dos dois, ambos assumem `Email`. Só é
permitido um método de notificação por signatário.

| Verificação | Notificação permitida | Exige | Custo por signatário |
| --- | --- | --- | --- |
| `models.VerificationMethodEmail` *(padrão)* | `Email` | E-mail no signatário | 0 créditos |
| `models.VerificationMethodWhatsApp` | `Whatsapp` | `whatsapp_phone_number` e assinatura paga | 0,45 crédito |
| `models.VerificationMethodDigitalCertificate` | `Email` ou `Whatsapp` | Recurso Certificado Digital, e CPF ou CNPJ no `government_id` do signatário | 2 créditos mais a notificação |

Nenhum método de verificação é cobrado por si — paga-se pela notificação com a qual
ele anda junto. Escolher verificação por WhatsApp escolhe também a notificação por
WhatsApp, e os 0,45 crédito dela. `DigitalCertificate` é a exceção: a assinatura em
si custa 2 créditos por signatário além da notificação, debitados na criação do
assignment. `client.Assignments.EstimateCostWithRequest` devolve o total exato antes
do compromisso, detalhado em `Breakdown`.

Uma combinação inválida é recusada com `400`, então use as constantes em vez de
literais de string.

### Assinando com certificado ICP-Brasil

Um signatário `DigitalCertificate` assina com o próprio certificado ICP-Brasil —
**A1** em software ou **A3** em token ou cartão — pela extensão de navegador Web PKI,
produzindo uma assinatura **PAdES qualificada** no documento.

Três condições valem. A conta precisa do recurso Certificado Digital (planos
Standard e Pro). O signatário precisa de CPF ou CNPJ em `government_id`, gravado por
`client.Signers.Update`: um CPF exige o certificado daquela pessoa (e-CPF, ou e-CNPJ
que a nomeie como representante legal), e um CNPJ exige um e-CNPJ daquela empresa. E
cada signatário por certificado precisa estar **sozinho no seu passo de assinatura**.

```go
_, err = client.Signers.Update(ctx, "", signer.ID, &models.UpdateSignerRequest{
	GovernmentID: &cpf,
})

assignment, err := client.Assignments.Create(ctx, document.ID, &models.CreateAssignmentRequest{
	Method: models.MethodVirtual,
	Signers: []models.SignerReference{{
		ID:                  signer.ID,
		VerificationMethod:  models.VerificationMethodDigitalCertificate,
		NotificationMethods: []string{models.NotificationMethodEmail},
		Step:                &primeiroPasso,
	}},
})
```

Esse signatário confirma os dados e aceita os termos normalmente — uma chamada a
`ConfirmDataAndGet` resolve as duas coisas —, mas `client.Assignments.Sign` **o
recusa** com `400`. A assinatura dele vem de uma troca em dois passos com a
extensão:

```go
start, err := client.Signers.StartCertificateSignature(ctx, signerAccessCode)
// Entregue start.Token à extensão Web PKI, que o assina com o certificado
// do signatário e devolve a assinatura.
result, err := client.Signers.CompleteCertificateSignature(ctx, signerAccessCode,
	&models.CompleteCertificateSignatureRequest{Token: start.Token, Signature: signature})
// result.SignerName é o nome lido do certificado.
```

Quando o documento fecha, o artefato `pades` carrega as assinaturas qualificadas.

> Essas duas rotas são citadas na referência mas não têm esquema de requisição nem
> de resposta publicado. O SDK implementa as formas que o próprio fluxo de
> assinatura da Assinafy usa; `CompleteCertificateSignatureRequest.Extra` envia
> qualquer campo adicional que uma implantação espere.

## Conectando a conta de outras pessoas com OAuth

Use OAuth quando você constrói um produto que **outros clientes da Assinafy conectam
à conta deles**. O usuário aprova sua aplicação uma vez e você recebe um token
limitado às permissões aprovadas e a uma única conta — você nunca lida com a senha
nem com a chave de API dele, e ele pode desconectar você quando quiser. Automatizar a
sua própria conta não precisa de nada disso; continue com uma chave de API.

Registre a aplicação no app da Assinafy, em **Configurações → Aplicações OAuth**.
É preciso ser proprietário da conta que vai possuí-la, e o plano dessa conta precisa
incluir aplicações OAuth. O registro gera um `client_id` e, para uma aplicação
**confidencial** — cujo código roda em um servidor seu —, um `client_secret`. Uma
aplicação **pública**, que roda no dispositivo do usuário e não guarda segredo, não
recebe nenhum e se autentica só com PKCE. Aplicações não são criadas pela API.

O pacote `oauth` implementa o fluxo inteiro sobre a biblioteca padrão.

**Passo 1 — inicie uma conexão.** Gere um par PKCE e um `state` novos a cada
tentativa, guarde os dois na sessão do usuário e mande o navegador para a URL com
uma navegação de página inteira:

```go
config := &oauth.Config{
	ClientID:     os.Getenv("ASSINAFY_CLIENT_ID"),
	ClientSecret: os.Getenv("ASSINAFY_CLIENT_SECRET"), // omita numa aplicação pública
	RedirectURI:  "https://meuapp.example/oauth/callback",
	Scopes: []string{
		oauth.ScopeDocumentsRead,
		oauth.ScopeDocumentsWrite,
		oauth.ScopeOfflineAccess,
	},
}

pkce, err := oauth.NewPKCE()
state, err := oauth.NewState()
// Guarde pkce.Verifier e state na sessão deste usuário.

authorizeURL, err := config.AuthorizationURL(oauth.AuthorizationRequest{
	State:         state,
	CodeChallenge: pkce.Challenge,
})
http.Redirect(w, r, authorizeURL, http.StatusFound)
```

**Passo 2 — trate o retorno.** `ParseCallback` confere o `state` em tempo constante
e o parâmetro `iss` antes de devolver qualquer coisa, e reporta uma recusa do usuário
como `*oauth.Error` com código `access_denied`. O código é de uso único e expira 60
segundos depois da aprovação, então troque-o na hora:

```go
code, err := config.ParseCallback(r.URL, sessionState)
if err != nil {
	// oauth.ErrorCode(err) == oauth.ErrCodeAccessDenied quando o usuário recusou
	return err
}
token, err := config.Exchange(ctx, code, sessionVerifier)
```

**Passo 3 — descubra a conta e chame a API.** Um token pertence à única conta que o
usuário escolheu, então `Accounts.List` devolve exatamente essa conta. Guarde o ID
dela junto do token:

```go
source := oauth.NewTokenSource(config, token, func(renewed *oauth.Token) error {
	return store.SaveTokens(userID, renewed) // antes de o novo token de acesso ser usado
})

client, err := assinafy.NewClient(assinafy.ClientOptions{TokenSource: source})
workspaces, err := client.Accounts.List(ctx)
client, err = assinafy.NewClient(assinafy.ClientOptions{
	TokenSource: source,
	AccountID:   workspaces[0].ID,
})
```

O token source renova um token de acesso vencido a partir do refresh token antes de
cada requisição. Tokens de acesso duram uma hora; uma conexão dura **30 dias a partir
da aprovação do usuário**, e renovar não estende esse prazo — conte com reconexões
mensais.

> **Toda renovação rotaciona o refresh token e aposenta o anterior.** Reapresentar um
> refresh token aposentado é indistinguível de um token roubado sendo reusado, então
> o servidor encerra a conexão inteira. Salve o token novo dentro de `onRefresh`
> antes de qualquer outra coisa — devolver um erro ali aborta a renovação, de modo que
> seu armazenamento nunca fica com um token morto —, trate um timeout como "talvez
> tenha funcionado" e releia o que você salvou, e renove uma conexão por vez.
> `TokenSource` serializa as próprias renovações, então chamadas concorrentes
> compartilham uma só.

**Permissões.** Peça o mínimo; cada permissão a mais é outra linha que o usuário lê
antes de decidir, e ele aprova tudo ou nada.

| Constante | Permite à sua aplicação |
| --- | --- |
| `oauth.ScopeDocumentsRead` | Ler documentos, seus signatários, assignments e atividades |
| `oauth.ScopeDocumentsWrite` | Criar documentos e enviá-los para assinatura — isso pode gastar créditos de notificação da conta |
| `oauth.ScopeTemplatesRead` | Ler modelos |
| `oauth.ScopeTemplatesWrite` | Criar e alterar modelos |
| `oauth.ScopeAccountRead` | Ler o perfil, o tema e o logotipo da conta |
| `oauth.ScopeWebhooksWrite` | Configurar e desativar a assinatura de webhooks da conta |
| `oauth.ScopeOpenID` | Receber um `id_token` identificando o usuário, e chamar `UserInfo` |
| `oauth.ScopeProfile` | Ler o nome do usuário |
| `oauth.ScopeEmail` | Ler o e-mail do usuário e se ele está verificado |
| `oauth.ScopeOfflineAccess` | Receber um refresh token, para seguir funcionando com o usuário ausente |

Leia `Token.Scope` para saber o que foi realmente concedido, em vez de supor.
Faturamento, membros da conta, gestão de credenciais e administração nunca são
alcançáveis por um token OAuth, quaisquer que sejam os escopos. Chamar um endpoint
sem o escopo necessário devolve `403` com um desafio que nomeia o que falta:

```go
if scope, ok := sdkerrors.InsufficientScope(err); ok {
	// Reconecte pedindo scope; repetir a chamada falharia igual.
}
```

Um `403` sem esse cabeçalho tem outra causa: outra conta, o papel do próprio usuário,
ou uma área que o OAuth não alcança. Um token vale só para a conta para a qual foi
emitido — chamar qualquer outra devolve `403`, mesmo uma da qual o mesmo usuário
participa —, então um cliente com várias contas conecta cada uma separadamente.

**Identificando o usuário.** Com `ScopeOpenID` o token traz um `id_token`, um JWT
RS256 dizendo quem aprovou. Valide-o com qualquer biblioteca OpenID Connect contra o
JWKS do emissor, e leia nome e e-mail em `config.UserInfo(ctx, accessToken)`.

**Desconectando.** Quando um usuário desconecta no seu produto, revogue o token em
vez de apenas apagá-lo. A revogação sempre responde sucesso:

```go
err := config.Revoke(ctx, refreshToken, oauth.HintRefreshToken)
```

**Descoberta.** As URLs são publicadas pelo servidor de autorização, e
`Config.Endpoint` já aponta para produção, então a maioria das integrações não
configura nada. Para lê-las em tempo de execução:

```go
resource, err := oauth.DiscoverProtectedResource(ctx, "", nil)          // esta API
metadata, err := oauth.DiscoverAuthorizationServer(ctx, resource.Issuer(), nil)
config.Endpoint = metadata.Endpoint()
```

Antes de ir para produção: par PKCE e `state` novos a cada tentativa; `state` e `iss`
conferidos; o segredo só no seu servidor; o refresh token rotacionado salvo antes do
uso; `401` tratado com renovação e, se ela falhar, pedindo reconexão ao usuário; toda
URI de redirecionamento registrada, `https://` e comparada caractere a caractere.
Aplicações novas não são verificadas — a tela de aprovação avisa, e elas conectam no
máximo 25 contas —, então peça verificação à Assinafy antes de ir além de um piloto.
Os endpoints de autorização e token aceitam 50 requisições por minuto por IP.

## Criando documentos a partir de modelos

Um modelo carrega campos já posicionados e papéis de assinatura nomeados, então gerar
um documento e o assignment dele é uma chamada só. Modelos são criados na aplicação
web da Assinafy; o SDK os lê e gera a partir deles.

```go
templates, err := client.Templates.List(ctx, "", nil)
template, err := client.Templates.Get(ctx, "", templates.Data[0].ID) // papéis, páginas, etiquetas padrão

estimate, err := client.Documents.EstimateCostFromTemplate(ctx, "", template.ID,
	[]models.TemplateSigner{{RoleID: template.Roles[0].ID, VerificationMethod: models.VerificationMethodEmail}})

document, err := client.Documents.CreateFromTemplate(ctx, "", template.ID,
	[]models.TemplateSigner{{RoleID: template.Roles[0].ID, ID: signer.ID}},
	&models.CreateDocumentFromTemplateOptions{
		Name: "Contrato da Exemplo Ltda",
		Tags: []string{"contratos"},
	})
```

Informe um `TemplateSigner` por papel, referenciando signatários que já existem na
conta. `CreateDocumentFromTemplateOptions.EditorFields` preenche os valores do editor
do modelo, e as `Tags` são somadas às etiquetas padrão do modelo.

## Organizando documentos com etiquetas e campos

**Etiquetas** são rótulos da conta. Os nomes são únicos por conta, sem diferenciar
maiúsculas, então criar uma duplicata devolve `409`.

```go
tag, err := client.Tags.Create(ctx, "", &models.CreateTagRequest{Name: "contratos"})
_, err = client.Documents.AppendTags(ctx, "", document.ID, []string{tag.ID})
_, err = client.Documents.ReplaceTags(ctx, "", document.ID, []string{tag.ID})
err = client.Documents.DetachTag(ctx, "", document.ID, tag.ID)
```

Filtre por etiqueta com `models.ListParams{Tags: []string{tagID}}`, que o SDK envia
como lista separada por vírgulas. O filtro é conjuntivo: o documento precisa ter
todas as etiquetas listadas. `client.Tags.Delete(ctx, "", tagID, force)` remove uma
etiqueta, desanexando-a dos documentos quando `force` é verdadeiro.

**Campos** são as entradas tipadas que um assignment `collect` pede ao signatário.
`client.Fields.ListTypes` devolve os tipos disponíveis; `client.Fields.Create`,
`Get`, `Update` e `Delete` gerenciam as definições da conta, e
`client.Fields.ValidateAuthenticated` e `ValidateMultipleAuthenticated` conferem
valores contra uma definição antes do envio.

## Recebendo webhooks

Aponte a Assinafy para um endpoint, escolha os eventos e trate as entregas.

```go
events, err := client.Webhooks.ListEventTypes(ctx)
_, err = client.Webhooks.UpdateSubscription(ctx, "", &models.UpdateWebhookSubscriptionRequest{
	Events:   []string{"document_ready", "signer_signed_document"},
	IsActive: true,
	URL:      "https://example.com/hooks/assinafy",
	Email:    "ops@example.com",
})
```

Os quatro campos são obrigatórios na API e sempre são enviados.
`client.Webhooks.Inactivate` é a forma documentada de parar as entregas sem descartar
a configuração.

Decodifique um corpo entregue com o verificador:

```go
verifier := assinafy.NewWebhookVerifier(sharedSecret)
event, err := verifier.ExtractEvent(requestBody)
```

`client.Webhooks.ListDispatches` devolve o histórico de entregas — código de status,
corpo da resposta e erro por tentativa — filtrável por evento, resultado e intervalo
Unix, e `client.Webhooks.RetryDispatch` força nova tentativa.

Deduplique por `WebhookPayload.ID`, aceite campos e nomes de evento desconhecidos, e
não dependa da ordem entre eventos. O catálogo completo, com sujeito, objeto e chaves
de payload de cada evento, está em [docs/API.md](docs/API.md#webhook-payloads).

`WebhookVerifier.Verify` implementa `hex(HMAC-SHA256(secret, body))`. O contrato
publicado não documenta nenhum cabeçalho de assinatura, então use `Verify` apenas
depois que a Assinafy confirmar esse esquema e esse cabeçalho para a sua integração.

## Contas, usuários e estatísticas

`client.Accounts` lê e administra contas: `List`, `Get`, `Create`, `Update`,
`Delete`, além de `GetTheme`, `DownloadLogo`, `UploadLogo` e `DeleteLogo` para a
identidade visual que o signatário vê. Apagar uma conta é destrutivo e devolve os
impedimentos em `APIError.Restrictions`, a menos que se passe `force`.

`client.Users` cobre o usuário autenticado: `GetSelf`, mais
`GetNotificationPreferences` e `UpdateNotificationPreferences` para as nove
notificações por e-mail do proprietário. Uma atualização mescla as chaves enviadas e
devolve o mapa completo.

`client.Accounts.Stats` e `client.Users.Stats` devolvem o funil de documentos —
enviados, solicitados, visualizados, concluídos, certificados — mensalmente (últimos
doze meses) ou diariamente para um mês `YYYY-MM`. As duas séries vêm preenchidas com
zeros.

`client.Authentication` cuida de `Login`, `SocialLogin`, `LinkSocialLogin`, dos
fluxos de senha e do ciclo `CreateAPIKey`/`GetAPIKey`/`DeleteAPIKey`. Criar uma chave
substitui a anterior e é a única vez em que a chave completa é devolvida.
`SocialLoginURL` monta a URL de navegador que inicia o login social de um provedor e
não faz requisição alguma.

## Erros e novas tentativas

Toda falha é de um de três tipos, e todos são inspecionáveis:

```go
document, err := client.Documents.Get(ctx, documentID)

var apiErr *sdkerrors.APIError
switch {
case sdkerrors.IsStatusCode(err, http.StatusNotFound):
	// o documento não existe
case stderrors.As(err, &apiErr):
	// qualquer outra resposta: apiErr.StatusCode, .Message, .Data, .Headers, .Restrictions
case sdkerrors.IsRetryable(err):
	// 429, 5xx ou falha de transporte — repita com backoff
case stderrors.Is(err, sdkerrors.ErrInvalidInput):
	// rejeitado localmente, antes de qualquer requisição
}
```

- `*errors.APIError` — resposta HTTP ou status de envelope de insucesso. Carrega
  `StatusCode`, `Message`, `Data` opcional, os `Headers` da resposta (inclusive
  `Retry-After`) e as `Restrictions` de exclusão de conta.
- `*errors.NetworkError` — falha de transporte. Qualquer `signer-access-code` na URL
  é ocultado antes de o erro ser devolvido.
- `errors.ErrInvalidInput` — rejeição local: um ID obrigatório vazio, um upload que
  não é PDF, um arquivo grande demais, um prazo malformado.

Numa conexão OAuth, `errors.InsufficientScope(err)` lê o escopo nomeado por um `403`
com `insufficient_scope`, e o pacote `oauth` devolve `*oauth.Error` com o código RFC
6749 — leia-o com `oauth.ErrorCode(err)`.

`errors.IsRetryable` responde `true` para HTTP 429, 5xx e falhas de transporte, e
`false` para contextos cancelados e redirecionamentos recusados. O SDK não repete
nada por conta própria: política de repetição, backoff e idempotência são de quem
chama, e várias operações aqui enviam e-mail ou gastam créditos.

`errors.ErrUnsafeRedirect` envolve um redirecionamento entre origens que o SDK se
recusou a seguir para uma requisição com corpo ou método não idempotente.
Redirecionamentos somente-leitura entre origens são seguidos com a chave de API, o
token bearer e o código de acesso do signatário removidos.

## Cuidando das credenciais

- Trate todo `signer-access-code` e toda URL de assinatura como credencial. Eles dão
  acesso à sessão de assinatura; nunca registre em log. O SDK os remove de requisições
  redirecionadas e os oculta em erros de rede.
- Nunca leve uma chave de API para um cliente de navegador ou celular. As chaves são
  permanentes e têm escopo do usuário que as criou. E nunca peça a chave de outro
  cliente da Assinafy — é para isso que existe o
  [OAuth](#conectando-a-conta-de-outras-pessoas-com-oauth).
- Um `client_secret` de OAuth só existe em um servidor seu, nunca em código de
  celular, de navegador ou em um repositório. Uma aplicação pública não recebe
  nenhum e se autentica com PKCE. Rotacionar um segredo perdido invalida o anterior
  imediatamente, e apagar a aplicação desconecta todos os usuários de uma vez.
- `BaseURL` precisa ser uma URL HTTP(S) absoluta sem credenciais, consulta ou
  fragmento embutidos; qualquer outra coisa é recusada com `ErrInvalidBaseURL`.
- Artefatos baixados voltam como bytes e nunca são gravados em disco pelo SDK.
  Escolha o caminho e as permissões — `0o600` para qualquer coisa que identifique um
  signatário.
- Reporte uma suspeita de vulnerabilidade por [SECURITY.md](SECURITY.md).

## Cobertura da API

[docs/API.md](docs/API.md) mapeia cada operação à sua chamada no SDK, com o modo de
autenticação, os campos de requisição, o payload de resposta, o comportamento de
paginação ou binário, os códigos de erro declarados e um link para a referência
oficial. Também expande todos os modelos de resposta, campo a campo.

| Área | Operações | Recurso do SDK |
| --- | ---: | --- |
| Contas | 10 | `client.Accounts` |
| Assignments | 7 | `client.Assignments` |
| Autenticação | 9 | `client.Authentication` |
| Documentos | 18 | `client.Documents` |
| Campos | 8 | `client.Fields` |
| OAuth | 4 | pacote `oauth` |
| Signatários | 5 | `client.Signers` |
| Assinatura | 17 | `client.PublicDocuments`, `client.Signers`, `client.Assignments`, `client.SignerDocuments` |
| Etiquetas | 4 | `client.Tags` |
| Modelos | 1 | `client.Templates` |
| Usuários | 4 | `client.Users` |
| Webhooks | 6 | `client.Webhooks` |
| **Total** | **93** | |

Além dessas operações o SDK também expõe `Client.UploadAndRequestSignatures`,
`client.Templates.Get` (uma leitura de modelo único que a referência não lista como
operação), `client.Authentication.SocialLoginURL`, a troca ICP-Brasil e o verificador
de webhooks. Métodos depreciados mantidos por compatibilidade de código estão no fim
de [docs/API.md](docs/API.md#additional-exported-methods).

**Rotas sem contrato publicado.** `POST /v1/signers/certificate/start` e `/complete`
são citadas na referência mas não declaram corpo de requisição, payload de resposta
nem conjunto de erros. `client.Signers.StartCertificateSignature` e
`CompleteCertificateSignature` implementam as formas que o próprio fluxo de
assinatura da Assinafy usa; trate-as como não verificadas e veja
[Assinando com certificado ICP-Brasil](#assinando-com-certificado-icp-brasil). Toda
outra rota alcançável em produção tem um método tipado aqui.

## Ambientes

| | |
| --- | --- |
| Produção | `https://api.assinafy.com.br/v1` |
| Sandbox | `assinafy.SandboxBaseURL` — `https://sandbox.assinafy.com.br/v1` |

O sandbox é gratuito e espelha a produção para testar a integração de ponta a ponta.
As quatro operações OAuth existem apenas em produção.

## Desenvolvimento

```bash
go build ./...
go test -race ./...
go vet ./...
gofmt -l .
golangci-lint run
```

O GitHub Actions roda as mesmas verificações — mais `govulncheck` e um piso de
cobertura — a cada push e pull request, com as revisões das actions fixadas por SHA
de commit. O repositório pode ser espelhado a partir do GitLab; o GitHub continua
sendo o alvo de execução da CI mostrado no selo acima.

### Testes de integração

Os testes `TestIntegration*` são pulados a menos que `ASSINAFY_RUN_INTEGRATION_TESTS=1`
e tanto `ASSINAFY_API_KEY` quanto `ASSINAFY_ACCOUNT_ID` estejam definidos. Eles usam
`https://sandbox.assinafy.com.br/v1` por padrão e recusam o host de produção a menos
que `ASSINAFY_RUN_PRODUCTION_TESTS=1` também esteja definido.

A suíte não é somente-leitura: cria, atualiza e apaga recursos temporários no
sandbox. Use uma conta descartável. Dois fluxos de maior impacto exigem opt-in
próprio.

| Variável | Finalidade |
| --- | --- |
| `ASSINAFY_RUN_INTEGRATION_TESTS=1` | Habilita a suíte de integração, que modifica dados. |
| `ASSINAFY_API_KEY` | Chave de API do sandbox; obrigatória. |
| `ASSINAFY_ACCOUNT_ID` | ID da conta no sandbox; obrigatório. |
| `ASSINAFY_BASE_URL` | Sobrescrita opcional; vazio usa `assinafy.SandboxBaseURL`. |
| `ASSINAFY_RUN_ASSIGNMENT_TESTS=1` | Roda o fluxo completo de assignment numa conta descartável, enviando e-mail real de solicitação, token público e reenvio. |
| `ASSINAFY_TEST_EMAIL_PRIMARY` | Primeiro destinatário, só em tempo de execução; obrigatório com os testes de assignment. |
| `ASSINAFY_TEST_EMAIL_SECONDARY` | Segundo destinatário, só em tempo de execução; obrigatório com os testes de assignment. |
| `ASSINAFY_RUN_ACCOUNT_ADMIN_TESTS=1` | Habilita o ciclo criar/atualizar/logotipo/apagar de conta. |
| `ASSINAFY_RUN_PRODUCTION_TESTS=1` | Remove a trava de segurança de produção. Não torna os testes somente-leitura. |

```bash
ASSINAFY_RUN_INTEGRATION_TESTS=1 ASSINAFY_API_KEY=... ASSINAFY_ACCOUNT_ID=... \
  go test -race -run '^TestIntegration' -v .
```

Habilite a execução em produção ou os fluxos opcionais apenas quando os efeitos
colaterais forem desejados. Operações que revogariam a credencial em uso, mudariam
uma senha ou forjariam um token de provedor social, de redefinição de senha ou de
acesso de signatário são cobertas por testes locais de método, caminho, autenticação,
requisição e resposta, em vez de chamadas reais.

## Documentação

- **[README.en.md](README.en.md)** — the same guide in English
- [Go Reference](https://pkg.go.dev/github.com/assinafy/golang-sdk)
- [Documentação da API](https://api.assinafy.com.br/v1/docs)
- [docs/API.md](docs/API.md) — mapa operação a operação e referência de payloads

## Licença

Distribuído sob a licença [MIT](LICENSE).
