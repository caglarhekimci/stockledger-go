# LinkedIn Türkiye teknoloji beklentileri — 3 Kasım 2026

Bu çalışma başvuru önerisi değil, iki portföy projesinin kapsamını belirleyen amaçlı bir ilan örneklemidir. Kamuya açık LinkedIn ilan sayfaları ve arama dizini incelendi. Tüm LinkedIn'e veya oturum açılmış arama sonuçlarına erişildiği iddia edilmez. Aramalarda teknoloji isimleri de kullanıldığından örneklem seçilim yanlılığı taşır; oranlar Türkiye geneline genellenemez.

**Güncellik sınırı:** İnceleme tarihi 2026-11-03, Europe/Istanbul. Arama aracının sunduğu sayfalar önbelleğe alınmış olabilir. Ayrıntı sayfasındaki kapanma ibaresi, arama özetine üstün tutuldu. Go örnekleminin 8/8'i kapalı; Next.js örnekleminin 6/8'i kapalı, 2/8'inde kapanma ibaresi görülmedi. Kapanma ibaresinin yokluğu aktif başvurunun bağımsız doğrulaması değildir. Dolayısıyla bu rapor güncel açık pozisyonların sayımı değildir; erişilebilir ilan metinlerindeki mühendislik beklentilerinin analizidir.

Tekil ilan kimliğiyle mükerrerler çıkarıldı; Azberry'nin aynı ailedeki ikinci ilanı sayılmadı. Go yalnız şirket adı olan sonuçlar, Türkiye dışına taşınma zorunluluğu ve İstanbul/Tehran konum çelişkisi olan sonuçlar çekirdek örnekleme alınmadı. Go'nun yalnız avantaj olduğu AppNation ve LOOP gibi ilanlar Go çekirdek sıklığına katılmadı.

**Kodlama:** R = aranan nitelik veya görev/sorumluluk; P = tercih/avantaj/alternatif teknoloji listesi; N = gözlenen metinde açıkça yok. R, her aday için kesin eleme koşulu anlamına gelmez; SQL aile şartı PostgreSQL'e özel zorunluluk olarak yorumlanmaz. Bir ilanda aynı beceri bir kez sayıldı. Yalnız ürün bağlamından teknoloji tahmin edilmedi.

## Go / Golang

Örneklem: 8 ilan.

| Beklenti | R | P | Toplam anılma |
|---|---:|---:|---:|
| SQL | 7 | 1 | 8/8 |
| Redis | 0 | 5 | 5/8 |
| Mesajlasma | 1 | 5 | 6/8 |
| Eszamanlilik | 5 | 1 | 6/8 |
| Docker | 5 | 2 | 7/8 |
| Test | 6 | 1 | 7/8 |
| Izlenebilirlik | 3 | 3 | 6/8 |
| API | 6 | 1 | 7/8 |

| ID | Şirket / ilan | Konum | Sayfa durumu |
|---|---|---|---|
| G1 | [Cimri — Golang Backend Developer](https://tr.linkedin.com/jobs/view/golang-backend-developer-at-cimri-3863565726) | İstanbul | kapalı |
| G2 | [Azberry BV — Golang Developer (Mid-Level) – High-Scale Systems](https://tr.linkedin.com/jobs/view/golang-developer-mid-level-%E2%80%93-high-scale-systems-at-azberry-bv-4366341387) | Türkiye | kapalı |
| G3 | [Getir — Senior Backend Engineer – Go](https://tr.linkedin.com/jobs/view/senior-backend-engineer-go-at-getir-4208598273) | İstanbul | kapalı |
| G4 | [Calico Teknoloji — Backend Engineer (Golang)](https://tr.linkedin.com/jobs/view/backend-engineer-golang-at-calico-teknoloji-4439226268) | Ankara | kapalı |
| G5 | [Kartaca — Go Developer](https://tr.linkedin.com/jobs/view/go-developer-at-kartaca-4461053757) | İstanbul | kapalı |
| G6 | [Ahlatcı Holding / Bixcod — Senior Backend Developer (Go / Golang)](https://tr.linkedin.com/jobs/view/senior-backend-developer-go-golang-at-ahlatc%C4%B1-holding-4352753574) | Üsküdar | kapalı |
| G7 | [FTech Labs — Software Engineer – Golang](https://tr.linkedin.com/jobs/view/software-engineer-golang-at-ftech-labs-bili%C5%9Fim-a-%C5%9F-3772464893) | Türkiye | kapalı |
| G8 | [Deloitte — Golang Developer](https://tr.linkedin.com/jobs/view/golang-developer-at-deloitte-4462290903) | İstanbul | kapalı |

### İlan bazında dayanak

- **G1:** PostgreSQL, test edilebilir kod ve ölçeklenebilir API sorumlulukları var. Redis ve Kafka tercih listesinde.
- **G2:** PostgreSQL, goroutine/channel ve Docker veya Kubernetes deneyimi isteniyor. Redis/Kafka artı; test araçları açıkça belirtilmiyor.
- **G3:** SQL/NoSQL, olay tabanlı mesajlaşma, test ve OpenTelemetry birlikte yer alıyor. Redis adı açıkça yok.
- **G4:** Gerçek zamanlı veri motoru; Go eşzamanlılığı, PostgreSQL/ClickHouse, ağ bilgisi, Docker, CI ve kapsamlı test altyapısı. Broker adı olmadığı için mesajlaşma sayılmadı.
- **G5:** Goroutine/channel, SQL/NoSQL, web servisleri ve test sorumluluğu var. Redis/RabbitMQ/Kafka tercih. Arama özeti açık görünse de ayrıntı sayfası kapalı dedi.
- **G6:** PostgreSQL transaction/concurrency, API, Docker ve CI bekleniyor. Redis, broker, gRPC ve izlenebilirlik artı listesinde.
- **G7:** Mikroservis, önbellek ve mesajlaşma prensipleri; PostgreSQL/gRPC ve Redis/SQS/Docker/test deneyimleri avantaj olarak ayrılmış.
- **G8:** Go backend, SQL, concurrency, Docker/Kubernetes, API doğrulama, logging ve test; ayrıca Java/Spring geçmişi isteniyor. Redis varsayılmadı.

### Denetlenebilir kodlama matrisi

| ID | SQL | Redis | Mesajlasma | Eszamanlilik | Docker | Test | Izlenebilirlik | API |
|---|---|---|---|---|---|---|---|---|
| G1 | R | P | P | P | P | R | P | R |
| G2 | R | P | P | R | R | N | P | N |
| G3 | R | N | R | N | R | R | R | R |
| G4 | R | N | N | R | R | R | R | R |
| G5 | R | P | P | R | N | R | N | R |
| G6 | R | P | P | R | R | R | P | R |
| G7 | P | P | P | N | P | P | N | P |
| G8 | R | N | N | R | R | R | R | R |

## Next.js

Örneklem: 8 ilan.

| Beklenti | R | P | Toplam anılma |
|---|---:|---:|---:|
| TypeScript | 7 | 1 | 8/8 |
| React | 8 | 0 | 8/8 |
| API | 6 | 1 | 7/8 |
| State | 5 | 0 | 5/8 |
| Render | 4 | 2 | 6/8 |
| Erisilebilirlik | 5 | 0 | 5/8 |
| Performans | 6 | 0 | 6/8 |
| Test | 4 | 2 | 6/8 |
| Tailwind | 2 | 2 | 4/8 |

| ID | Şirket / ilan | Konum | Sayfa durumu |
|---|---|---|---|
| N1 | [zevnika — Frontend Developer (React / Next.js)](https://tr.linkedin.com/jobs/view/frontend-developer-react-next-js-at-zevnika-4483058218) | İstanbul | kapanma ibaresi yok |
| N2 | [TEXIT — Frontend Developer](https://tr.linkedin.com/jobs/view/frontend-developer-at-tex%C4%B1t-4339523608) | Ankara | kapalı |
| N3 | [Oculon Systems — Frontend Developer: React / Next.js / TypeScript](https://tr.linkedin.com/jobs/view/frontend-developer-react-next-js-typescr%C4%B1pt-at-oculon-systems-4475307427) | Türkiye | kapalı |
| N4 | [LOOP Digital Signage — Full Stack Engineer](https://tr.linkedin.com/jobs/view/full-stack-engineer-at-loop-digital-signage-4479438399) | İstanbul | kapanma ibaresi yok |
| N5 | [Mytech Teknoloji — Next.js Geliştirici](https://tr.linkedin.com/jobs/view/next-js-geli%C5%9Ftirici-at-mytech-teknoloji-yaz%C4%B1l%C4%B1m-anonim-%C5%9Firketi-4282808236) | İstanbul | kapalı |
| N6 | [Alpaco — Frontend Developer (Next.js / React)](https://tr.linkedin.com/jobs/view/frontend-developer-next-js-react-at-alpaco-4379848576) | İstanbul | kapalı |
| N7 | [EKOS.AI — Frontend Engineer (React/Next.js – Pages Router)](https://tr.linkedin.com/jobs/view/frontend-engineer-react-next-js-%E2%80%93-pages-router-at-ekos-a%C4%B1-4265468882) | Türkiye / İstanbul | kapalı |
| N8 | [Wearon Studio — Full Stack Developer](https://tr.linkedin.com/jobs/view/full-stack-developer-at-wearon-studio-4339764305) | Pendik | kapalı |

### İlan bazında dayanak

- **N1:** TypeScript, REST, state araçları, SSR/SSG/App Router ve erişilebilirlik. Otomatik test tercih. Açık olduğu yalnızca kapanma ibaresi yokluğundan kesinleştirilmedi.
- **N2:** App Router, server/client sınırı, TanStack Query, auth ve form doğrulaması; code review ve temel test süreçleri. Tailwind destekleyici/alternatif araç.
- **N3:** Modüler mimari, API sözleşmesi, state, boş/yükleniyor/hata durumları, erişilebilirlik ve test. App Router/RSC avantaj listesinde.
- **N4:** Next.js/React TypeScript, Node REST, MongoDB, cloud ve test/CI. Medya pipeline/cihaz güvenilirliği özel alan beklentisi.
- **N5:** App Router, RSC, Server Actions/Route Handler, caching ve render modları zorunlu. TypeScript ve test araçları tercih başlığında.
- **N6:** TypeScript, REST/GraphQL, state, yeniden kullanılabilir UI ve test süreçleri. SSR/SSG ve tasarım sistemi artı.
- **N7:** Pages Router, SSR/CSR, TanStack Query, Tailwind/MUI, bileşen mimarisi ve erişilebilirlik. Modern Next.js her ilanda yalnız App Router demek değil.
- **N8:** Next.js/TypeScript ve PostgreSQL merkezli ürün sahipliği. Supabase ve görsel/video model API deneyimi tercih. React veya Next.js alternatifli anlatılıyor.

### Denetlenebilir kodlama matrisi

| ID | TypeScript | React | API | State | Render | Erisilebilirlik | Performans | Test | Tailwind |
|---|---|---|---|---|---|---|---|---|---|
| N1 | R | R | R | R | R | R | R | P | R |
| N2 | R | R | R | R | R | R | N | R | P |
| N3 | R | R | R | R | P | R | R | R | P |
| N4 | R | R | R | N | N | N | R | R | N |
| N5 | P | R | R | N | R | R | R | P | N |
| N6 | R | R | R | R | P | N | R | R | N |
| N7 | R | R | N | R | R | R | R | N | R |
| N8 | R | R | P | N | N | N | N | N | N |

## Proje kararları

**Go:** SQL tüm 8 metinde; test 7/8, Docker 7/8, API 7/8, concurrency/parallelism 6/8, broker/mesajlaşma örnekleri 6/8. Redis 5/8'de anılıyor, bu beşinin tamamı tercih/avantaj grubunda; “çoğu Go ilanı Redis'i zorunlu tutuyor” sonucu çıkarılamaz. StockLedger bu beklentileri stok atomikliği, idempotency, transaction/outbox, Redis limiter, Kafka worker ve operasyon testleriyle gösterir. Yalnız etiket eklemek yerine veri bütünlüğü problemi seçildi.

**Next.js:** TypeScript 8/8 (7 R, 1 P); React/Next ekosistemi 8/8; API ile ilgili çalışma 7/8; render sınırları 6/8; test 6/8; performans 6/8; state 5/8; erişilebilirlik 5/8. Tailwind 4/8'de örnek/alternatif olarak geçer; zorunlu ortak standart sayılmaz. ProcureFlow bunları App Router server/client ayrımı, tipli API, form doğrulama, TanStack Query, erişilebilir bileşenler ve gerçek tarayıcı iş akışıyla gösterir. SQLite portföyün kolay çalışması için mimari tercihtir; ilanlarda PostgreSQL yaygınlığı iddiası değildir.

İki proje mimari seçenekler, SOLID eşleştirmesi, güvenlik sınırları, test kanıtı ve kurulumu kendi belgelerinde açıklar. Bu projeler bir şirkette üretim deneyimi veya gerçek trafik kanıtı olarak sunulmamalıdır.
