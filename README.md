# SDK Go da Assinafy

*Português · [Read in English](README.en.md)*

[![CI](https://github.com/assinafy/golang-sdk/actions/workflows/ci.yml/badge.svg)](https://github.com/assinafy/golang-sdk/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/assinafy/golang-sdk/v2.svg)](https://pkg.go.dev/github.com/assinafy/golang-sdk/v2)

Cliente Go sem dependências de terceiros para as 147 operações publicadas da
[API Assinafy v1](https://api.assinafy.com.br/v1/docs). Requer Go 1.26+, TLS 1.2+
e usa a biblioteca padrão. A referência completa de métodos, campos de requisição,
respostas, autenticação e erros está em [docs/API.md](docs/API.md).

## Instalação e cliente

```bash
go get github.com/assinafy/golang-sdk/v2
```

`NewClient` aceita `APIKey`, `Token` ou `TokenSource`, nessa ordem de precedência.
`AccountID` é a conta padrão e `Timeout` vale 30 segundos quando não informado.
`BaseURL` usa produção; escolha `SandboxBaseURL` para testes. URLs externas precisam
ser HTTPS; HTTP é aceito somente em loopback para testes locais.

```go
client, err := assinafy.NewClient(assinafy.ClientOptions{
	APIKey: os.Getenv("ASSINAFY_API_KEY"), AccountID: os.Getenv("ASSINAFY_ACCOUNT_ID"),
	BaseURL: assinafy.SandboxBaseURL,
})
```

O cliente é seguro para uso concorrente. Métodos recebem `context.Context`,
desembrulham `{status,message,data}`, devolvem `*errors.APIError` para respostas de
falha, `*errors.NetworkError` para transporte e erros que envolvem
`errors.ErrInvalidInput` para rejeições locais.

## Programa completo: PDF até documento certificado

Defina `ASSINAFY_API_KEY`, `ASSINAFY_ACCOUNT_ID`, `ASSINAFY_PDF_PATH`,
`ASSINAFY_SIGNER_NAME` e `ASSINAFY_SIGNER_EMAIL`, então execute `go run main.go`.
O exemplo usa o sandbox, envia uma notificação real e espera a assinatura por até
30 minutos. Use um endereço `example.com` somente quando não quiser entrega.

```go
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	assinafy "github.com/assinafy/golang-sdk/v2"
	"github.com/assinafy/golang-sdk/v2/models"
)

func must[T any](value T, err error) T {
	if err != nil { log.Fatal(err) }
	return value
}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	client := must(assinafy.NewClient(assinafy.ClientOptions{
		APIKey: os.Getenv("ASSINAFY_API_KEY"), AccountID: os.Getenv("ASSINAFY_ACCOUNT_ID"),
		BaseURL: assinafy.SandboxBaseURL,
	}))
	pdf := must(os.ReadFile(os.Getenv("ASSINAFY_PDF_PATH")))
	document := must(client.Documents.Upload(ctx, "", pdf, "document.pdf", nil))

	email := os.Getenv("ASSINAFY_SIGNER_EMAIL")
	signer := must(client.Signers.Create(ctx, "", &models.CreateSignerRequest{
		FullName: os.Getenv("ASSINAFY_SIGNER_NAME"), Email: &email,
	}))
	estimate := must(client.Assignments.EstimateCostWithRequest(ctx, document.ID,
		&models.EstimateAssignmentCostRequest{
			Method: models.MethodVirtual,
			Signers: []models.EstimateAssignmentCostSigner{{
				VerificationMethod: models.VerificationMethodEmail,
			}},
		}))
	if !estimate.HasSufficientResources { log.Fatal(estimate.Message) }

	assignment := must(client.Assignments.Create(ctx, document.ID,
		&models.CreateAssignmentRequest{
			Method: models.MethodVirtual, MaxTotalCredits: &estimate.TotalCredits,
			Signers: []models.SignerReference{{
				ID: signer.ID, VerificationMethod: models.VerificationMethodEmail,
				NotificationMethods: []string{models.NotificationMethodEmail},
			}},
		}))
	fmt.Printf("document=%s signer=%s assignment=%s\n", document.ID, signer.ID, assignment.ID)

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for {
		document = must(client.Documents.Get(ctx, document.ID))
		switch document.Status {
		case models.StatusCertificated:
			pdf = must(client.Documents.Download(ctx, document.ID, "certificated"))
			if err := os.WriteFile("document-signed.pdf", pdf, 0o600); err != nil { log.Fatal(err) }
			return
		case models.StatusExpired, models.StatusRejectedBySigner,
			models.StatusRejectedByUser, models.StatusFailed:
			log.Fatalf("signing ended with status %q", document.Status)
		}
		select {
		case <-ctx.Done(): log.Fatal(ctx.Err())
		case <-ticker.C:
		}
	}
}
```

Guarde cada ID assim que for criado: upload, criação de signatário e convite não são
transacionais e não devem ser repetidos após um resultado de rede incerto. Reutilize
um signatário existente porque o e-mail é único na conta. Assignments `collect`
exigem páginas e campos com coordenadas de 150 DPI; o fluxo `virtual` não exige.

## Verificação, notificação e assinatura

- `Email`: notificação e OTP por e-mail.
- `Whatsapp`: notificação e OTP por WhatsApp; o sandbox pode responder 403 conforme o plano.
- `IdDocumentFaceMatchBasic`: captura biométrica hospedada, CPF válido, consentimento
  explícito e revisão pela conta quando o resultado fica `InReview`.
- `DigitalCertificate`: ICP-Brasil A1/A3 via Web PKI, CPF correspondente, assignment
  virtual e signatário sozinho no passo; custa 0,5 crédito mais a notificação.

Links de `Assignment.SigningURLs` permitem pedir o convite por e-mail; WhatsApp e
signatários embedded podem retornar URL nula (string vazia no SDK). Uma experiência própria
usa `GetSigningInfo`, `GetSelf`, `VerifyOTP`, `ConfirmDataAndGet`,
`AcceptTermsOnly`, `UploadSignatureWithReuse` e `Assignments.Sign`, sempre com o
`signer-access-code`. Biometria usa `GetIdentityVerification` e
`StartIdentityVerification`. Certificados usam `StartCertificateSignature` e
`CompleteCertificateSignature`; os dois payloads ainda não fazem parte da descrição
OpenAPI publicada. O SDK nunca mistura o código do signatário com credenciais da conta.

## Envelopes e assinatura incorporada

`client.Envelopes` reúne até 20 documentos por convite, com custo estimado,
`MaxTotalCredits` e `IdempotencyKey`. Faça upload, espere `metadata_ready`, estime,
crie e acompanhe cada documento até `certificated`; baixe o ZIP do envelope.
`client.Embed` registra origens HTTPS e cria sessões de cinco minutos para iframe,
com signatário `Embedded: true` e verificação Email ou WhatsApp. OAuth não cria essas
sessões. Carregue a URL devolvida sem registrá-la; cada abertura exige nova sessão.
[Payloads e fluxos](docs/API.md#envelope-request-and-response-payloads).

Criação por template com `IdempotencyKey` delega validação e recuperação de tentativas à API.

## OAuth de marketplace

Use `oauth.Config` com autorização por código, PKCE S256 e um `state` novo por
tentativa. Valide `state` e `iss`, troque o código imediatamente, salve o token antes
do uso e conecte o `oauth.TokenSource` ao `Client`. Refresh tokens rotacionam: salve o
novo token antes de expô-lo, serialize renovações por conexão e use uma trava no
armazenamento quando houver várias instâncias. `oauth.ErrRefreshIndeterminate` exige
reconexão, salvo quando o armazenamento já contém um token diferente e mais recente.
Na desconexão, pare chamadas, espere a renovação, revogue o último refresh token salvo
e só então apague-o. Valide `id_token` RS256 com o JWKS do emissor; o SDK não faz isso.

Peça somente os escopos necessários. `offline_access` habilita refresh; documentos,
modelos, conta, webhooks, OpenID, perfil e e-mail têm constantes próprias. Descoberta
fica em `DiscoverProtectedResource` e `DiscoverAuthorizationServer`, validando
`resource` e `issuer` sem redirecionamentos. Endpoints externos exigem HTTPS.

## Webhooks e operação segura

Crie o endpoint com `SigningEnabled`, persista o segredo `whsec_` devolvido uma única
vez e valide o corpo bruto com `NewWebhookVerifier(secret).VerifyRequest(headers,
body)`. Deduplique por `webhook-id`, responda `2xx` rapidamente e processe depois.
Rotação invalida o segredo anterior; `KeepPrevious: true` permite sobreposição por
24 horas. A API repete falhas transitórias e mantém sete
dias de histórico; veja os estados, tentativas e payloads em
[docs/API.md#webhook-payloads](docs/API.md#webhook-payloads).

Não repita mutações, uploads ou notificações automaticamente. Leituras podem usar
backoff quando `errors.IsRetryable` for verdadeiro. Trate URLs de assinatura,
`signer-access-code`, tokens e chaves como credenciais; nunca os registre. Downloads
voltam como `[]byte` e devem ser gravados com permissões restritas.

## Referência e desenvolvimento

- [docs/API.md](docs/API.md): 147 operações, payloads completos, paginação e binários.
- [Go Reference](https://pkg.go.dev/github.com/assinafy/golang-sdk/v2): todos os símbolos exportados.
- [SECURITY.md](SECURITY.md): comunicação privada de vulnerabilidades.
- Modelos, contratos, campos, templates, atividades, MFA e preferências estão na
  referência completa; o README mantém apenas o fluxo principal executável.

```bash
go build ./... && go vet ./... && go test -race ./...
```

Licença: [MIT](LICENSE).
