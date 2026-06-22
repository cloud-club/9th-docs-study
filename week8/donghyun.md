# 쿠버네티스 PKI / CSR / CA 스터디 문서

## 0. 전체 그림

쿠버네티스는 클라이언트(kubectl, kubelet 등)가 API 서버에 인증할 때 **공개키 기반 구조(PKI)** 와 **mTLS** 를 사용한다. 핵심 흐름은 다음과 같다.

```
개인키 생성 → CSR 생성 → CA 서명 → 인증서 발급 → TLS 핸드쉐이크에서 신원 증명
```

이 문서는 위 각 단계를 수학적 원리 → 자료구조 → 쿠버네티스 구현 순으로 다룬다.

---

## 1. RSA 공개키/개인키 수학

큰 소수 `p`, `q` 선택 → `n = p × q`, `φ(n) = (p-1)(q-1)`

```
공개키: (n, e)   ← e는 보통 65537
개인키: (n, d)   ← d × e ≡ 1 (mod φ(n))

서명:   sig = hash(m)^d mod n     (개인키로)
검증:   hash(m) == sig^e mod n     (공개키로)
```

핵심 성질: **개인키로 서명한 값은 대응하는 공개키로만 검증된다.** 이 비대칭성이 인증서·CSR·mTLS 전체의 신뢰 기반이다.

> 쿠버네티스 CA는 RSA 2048, 사용자 키 예시는 RSA 3072(`openssl genrsa -out myuser.key 3072`)를 쓴다. 서비스 인증서 예시(cfssl)는 ECDSA P-256을 쓰기도 한다.

---

## 2. X.509 인증서 구조

인증서는 ASN.1 DER로 인코딩된 구조체다.

```
Certificate ::= {
  TBSCertificate {            ← "To Be Signed" 본문
    version,
    serialNumber,
    issuer (CA의 DN),         ← 누가 서명했는지
    validity (notBefore, notAfter),
    subject (CN=myuser),      ← 누구의 인증서인지
    subjectPublicKeyInfo,     ← 이 주체의 공개키
    extensions                ← keyUsage, basicConstraints 등
  },
  signatureAlgorithm,
  signature                   ← CA가 TBSCertificate를 자기 개인키로 서명한 값
}
```

의미: 인증서 = **"이 subject의 공개키는 이것이다"를 CA가 서명으로 보증한 문서.**

### Issuer vs Subject

| 종류 | Issuer | Subject | 서명 주체 |
|------|--------|---------|-----------|
| 일반 인증서 | CA의 DN | 신청자 DN | CA 개인키 |
| CA 인증서 (self-signed) | 자기 자신 | 자기 자신 | 자기 개인키 |

`basicConstraints: CA:TRUE` 확장이 있어야 그 인증서가 다른 인증서를 서명할 수 있는 CA로 동작한다.

---

## 3. CSR (PKCS#10) 구조

```
CertificationRequest ::= {
  CertificationRequestInfo {
    version,
    subject (CN=myuser, O=group),  ← CN=사용자명, O=그룹
    subjectPublicKeyInfo,          ← 내 공개키
    attributes
  },
  signatureAlgorithm,
  signature                        ← 내 개인키로 CertificationRequestInfo를 서명한 값
}
```

### CSR에 자기 서명이 들어가는 이유

CA가 CSR을 받으면 "이 공개키가 진짜 신청자의 것인가?"를 검증해야 한다. CSR 안의 공개키로 CSR의 서명을 검증해서 통과하면, 신청자가 **대응하는 개인키를 실제로 보유**함이 수학적으로 증명된다. 개인키 없이는 유효한 서명을 만들 수 없기 때문이다 (PoP, Proof of Possession).

### CN / O 의 RBAC 매핑

- `CN` → 쿠버네티스 사용자명(user)
- `O` → 쿠버네티스 그룹(group)

API 서버는 클라이언트 인증서에서 이 값을 추출해 RBAC 인가 주체로 사용한다.

---

## 4. CA의 서명 과정

```
① CSR 수신
② CSR 내 공개키로 CSR 서명 검증     → 개인키 보유 확인 (PoP)
③ TBSCertificate 생성
     subject       ← CSR의 subject 그대로
     publicKey     ← CSR의 공개키 그대로
     issuer        ← CA 자신의 DN
     serialNumber, validity 등 CA가 채워넣음
④ hash = SHA-256(TBSCertificate)
⑤ sig  = hash^(d_CA) mod n_CA       (CA 개인키로 서명)
⑥ Certificate = TBSCertificate + sig  → 발급 완료
```

신청자의 공개키와 subject는 그대로 옮기고, issuer·serial·validity 같은 메타데이터는 CA가 채운 뒤 전체를 CA 개인키로 서명한다.

---

## 5. 쿠버네티스 CA 자체 구축

쿠버네티스 클러스터 CA는 외부 기관(DigiCert, Let's Encrypt 등)이 아니라 **클러스터 내부에 자체 구축**된다. `kubeadm init` 시 자동 생성된다.

### 생성 위치

```
/etc/kubernetes/pki/
├── ca.crt              ← Cluster CA 인증서 (self-signed, 10년)
├── ca.key              ← Cluster CA 개인키
├── etcd/
│   ├── ca.crt          ← etcd 전용 CA
│   └── ca.key
├── front-proxy-ca.crt  ← 확장 API 서버용 CA
└── front-proxy-ca.key
```

### kubeadm init이 CA를 만드는 과정

```
① RSA 2048 키쌍 생성
② self-signed X.509 인증서 생성
     Subject: CN=kubernetes
     validity: 10년
     basicConstraints: CA:TRUE
③ /etc/kubernetes/pki/ 에 저장
```

### self-signed의 의미

CA 인증서는 `Issuer == Subject`로 자기 자신을 서명한다. 외부 신뢰 체인이 없으므로 **인터넷에서는 신뢰받지 않고, 클러스터 내부에서만 신뢰 체계가 성립**한다.

---

## 6. CertificateSigningRequest 리소스와 승인

> 주의: 파일로 만든 PKCS#10 CSR과, 쿠버네티스 `CertificateSigningRequest` API 객체는 **다른 것**이다. PKCS#10 CSR을 base64 인코딩해서 CSR 객체의 `spec.request` 필드 안에 담는다.

### 클라이언트 인증서용 CSR 객체

```yaml
apiVersion: certificates.k8s.io/v1
kind: CertificateSigningRequest
metadata:
  name: myuser
spec:
  request: <base64(myuser.csr)>
  signerName: kubernetes.io/kube-apiserver-client
  expirationSeconds: 86400        # 최소 600초(10분) 이상
  usages:
  - client auth                   # 클라이언트 인증서는 반드시 client auth
```

### 승인 → 서명 주체

```
kubectl certificate approve myuser
```

승인되면 실제 서명은 **kube-controller-manager**가 수행한다.

```
kube-controller-manager 실행 옵션:
  --cluster-signing-cert-file=/etc/kubernetes/pki/ca.crt
  --cluster-signing-key-file=/etc/kubernetes/pki/ca.key
```

controller-manager가 `ca.key`로 CSR에 서명하여 인증서를 발급하고, 결과는 `.status.certificate`에 base64로 들어간다.

### 승인자의 책임 (보안)

승인자(사람 또는 컨트롤러)는 두 가지를 확인해야 한다.

1. **개인키 보유 확인** — CSR이 그에 대응하는 개인키로 서명되었는가 (제3자 사칭 방지)
2. **권한 확인** — 신청자가 요청한 신원/상황에서 동작할 자격이 있는가 (무단 클러스터 합류 방지)

CSR 승인 권한은 곧 "누가 누구를 신뢰할지 결정하는 권한"이므로 넓게/가볍게 부여하면 안 된다.

---

## 7. 클라이언트 인증서 발급 전체 절차 (사용자 인증)

```bash
# 1. 개인키 생성 (비밀 유지 — 가진 사람이 사용자를 사칭 가능)
openssl genrsa -out myuser.key 3072

# 2. PKCS#10 CSR 생성 (CN=사용자명)
openssl req -new -key myuser.key -out myuser.csr -subj "/CN=myuser"

# 3. CSR 객체 생성 (request에 base64(myuser.csr))
cat myuser.csr | base64 | tr -d "\n"
#   → 위 yaml의 spec.request에 넣어 kubectl apply

# 4. 승인 (controller-manager가 ca.key로 서명)
kubectl get csr
kubectl certificate approve myuser

# 5. 발급된 인증서 추출
kubectl get csr myuser -o jsonpath='{.status.certificate}' | base64 -d > myuser.crt

# 6. kubeconfig 등록
kubectl config set-credentials myuser \
  --client-key=myuser.key --client-certificate=myuser.crt --embed-certs=true
kubectl config set-context myuser --cluster=kubernetes --user=myuser

# 7. 확인
kubectl --context myuser auth whoami

# 8. RBAC 권한 부여
kubectl create role developer \
  --verb=create --verb=get --verb=list --verb=update --verb=delete --resource=pods
kubectl create rolebinding developer-binding-myuser \
  --role=developer --user=myuser
```

---

## 8. 서비스/파드용 TLS 인증서 (서버 인증)

서버 TLS 인증서는 커스텀 서명자와 직접 만든 CA로 발급한다.

```bash
# 1. cfssl로 개인키 + CSR 생성 (hosts에 서비스 DNS/IP 포함)
cat <<EOF | cfssl genkey - | cfssljson -bare server
{
  "hosts": ["my-svc.my-namespace.svc.cluster.local", "192.0.2.24"],
  "CN": "my-pod.my-namespace.pod.cluster.local",
  "key": { "algo": "ecdsa", "size": 256 }
}
EOF
# → server.csr, server-key.pem 생성

# 2. CSR 객체 생성 (커스텀 signerName, server auth)
#    signerName: example.com/serving
#    usages: digital signature, key encipherment, server auth

# 3. 승인
kubectl certificate approve my-svc.my-namespace

# 4. 자체 CA 생성
cat <<EOF | cfssl gencert -initca - | cfssljson -bare ca
{ "CN": "My Example Signer", "key": { "algo": "rsa", "size": 2048 } }
EOF
# → ca.pem, ca-key.pem 생성

# 5. CA로 CSR 서명
kubectl get csr my-svc.my-namespace -o jsonpath='{.spec.request}' \
  | base64 --decode \
  | cfssl sign -ca ca.pem -ca-key ca-key.pem -config server-signing-config.json - \
  | cfssljson -bare ca-signed-server

# 6. 서명된 인증서를 status에 업로드
kubectl get csr my-svc.my-namespace -o json \
  | jq '.status.certificate = "'$(base64 ca-signed-server.pem | tr -d '\n')'"' \
  | kubectl replace --raw /apis/certificates.k8s.io/v1/certificatesigningrequests/my-svc.my-namespace/status -f -

# 7. Secret / ConfigMap 으로 배포
kubectl get csr my-svc.my-namespace -o jsonpath='{.status.certificate}' | base64 --decode > server.crt
kubectl create secret tls server --cert server.crt --key server-key.pem
kubectl create configmap example-serving-ca --from-file ca.crt=ca.pem
```

---

## 9. mTLS 핸드쉐이크 (API 인증 시점)

kubectl이 kube-apiserver에 붙을 때 TLS 클라이언트 인증이 일어난다.

```
Client (kubectl)                    Server (kube-apiserver)
─────────────────                   ──────────────────────
ClientHello          ─────────────►
                     ◄─────────────  ServerHello
                                     Certificate (서버 인증서)
                                     CertificateRequest
Certificate          ─────────────►  (클라이언트 인증서 전송)
CertificateVerify    ─────────────►  (핸드쉐이크 메시지들을
                                      클라이언트 개인키로 서명한 값)
                                     ① 클라이언트 인증서 서명을
                                       cluster CA 공개키로 검증
                                     ② CertificateVerify를
                                       인증서 안의 공개키로 검증
                                     ③ CN/O 추출 → RBAC 인가 조회
Finished             ◄────────────►  Finished
```

### CertificateVerify가 필요한 이유

인증서만 전송하면 탈취한 인증서일 수 있다. CertificateVerify는 그 시점까지의 핸드쉐이크 메시지를 클라이언트 개인키로 서명한 값이라서, **실시간으로 개인키 보유를 증명**한다. 인증서(신원 주장) + CertificateVerify(개인키 보유 증명)가 합쳐져야 인증이 성립한다.

---

## 10. 두 인증서 타입 비교

| 구분 | 클라이언트 인증서 | 서버 TLS 인증서 |
|------|------------------|----------------|
| 대상 | kubectl 사용자, kubelet 등 | 서비스 / 파드 |
| signerName | `kubernetes.io/kube-apiserver-client` | 커스텀 (`example.com/serving`) |
| usages | `client auth` | `server auth` (+ digital signature, key encipherment) |
| 서명 주체 | kube-controller-manager (`ca.key`) | 직접 만든 CA |
| 결과물 활용 | kubeconfig에 등록 | Secret / ConfigMap으로 파드에 주입 |
| 키 알고리즘 예시 | RSA 3072 | ECDSA P-256 |

---

## 11. 직접 확인 명령어 (자체 클러스터)

```bash
# CA 인증서 내용
openssl x509 -in /etc/kubernetes/pki/ca.crt -text -noout | grep -E "Issuer|Subject|Not|CA"

# self-signed 검증 (Issuer == Subject 면 통과)
openssl verify -CAfile /etc/kubernetes/pki/ca.crt /etc/kubernetes/pki/ca.crt

# 발급한 사용자 인증서 검증
openssl verify -CAfile /etc/kubernetes/pki/ca.crt myuser.crt

# kubeconfig에 박힌 CA 확인
kubectl config view --raw -o jsonpath='{.clusters[0].cluster.certificate-authority-data}' | base64 -d | openssl x509 -text -noout
```

---

## 핵심 한 줄 요약

- **CSR** = 내 공개키 + "대응 개인키를 보유함"을 서명으로 증명한 발급 요청서
- **CA** = CSR 검증 후 자기 개인키로 서명해 인증서를 발급하는 주체 (k8s에선 self-signed로 자체 구축, `kubeadm init`이 생성)
- **인증서** = subject의 공개키를 CA가 서명으로 보증한 문서
- **mTLS** = 인증서(신원) + CertificateVerify(개인키 실시간 증명)로 API 서버에 인증