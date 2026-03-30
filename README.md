# 🛵 ordercli — Historie objednávek v terminálu

Fork [steipete/ordercli](https://github.com/steipete/ordercli) s přidanou podporou pro **Českou republiku (Foodora CZ)**.

Podporované platformy:
- `foodora` (funkční) — HU, SK, DE, AT, **CZ** ✅
- `deliveroo` (rozpracováno; vyžaduje `DELIVEROO_BEARER_TOKEN`)

Dostupné příkazy:
- `history` — historie objednávek
- `orders` — aktivní objednávky
- `order` / `history show` — detail objednávky
- `reorder` — opakování objednávky

---

## Instalace

### Požadavky

- [Go 1.24+](https://go.dev/dl/)

### Build
```sh
git clone https://github.com/Lukynnnn/ordercli.git
cd ordercli
go build ./cmd/ordercli
```

---

## 🇨🇿 Návod pro Českou republiku

### 1. Nastav CZ region
```sh
./ordercli foodora config set --country CZ
./ordercli foodora config show
```

### 2. Přihlášení

Foodora CZ vyžaduje `client_secret` a `client_id: iphone`. Přihlášení probíhá přes SMS OTP.
```sh
export FOODORA_CLIENT_SECRET='J7HNDia3f0paaJpXUadW8vi3rFKnYj48797QIF4HiYLF74aqoE'
./ordercli foodora login \
  --email tvuj@email.cz \
  --client-id iphone \
  --password heslo \
  --otp KOD_ZE_SMS
```

> ⚠️ `--otp` musíš zadat ihned po přijetí SMS — kód expiruje rychle. Nejlepší postup:
> 1. Spusť příkaz bez `--otp` → přijde SMS
> 2. Znovu spusť příkaz s `--otp XXXXXX`

### 3. Ověření
```sh
./ordercli foodora history
```

Měl bys vidět historii svých objednávek.

---

## Použití
```sh
# Historie objednávek
./ordercli foodora history
./ordercli foodora history --limit 50

# Detail objednávky
./ordercli foodora history show <orderCode>
./ordercli foodora history show <orderCode> --json

# Aktivní objednávky
./ordercli foodora orders
./ordercli foodora orders --watch

# Reorder — pouze náhled (nic neobjedná)
./ordercli foodora reorder <orderCode>

# Reorder — přidá do košíku (neodešle objednávku)
./ordercli foodora reorder <orderCode> --confirm

# Reorder s konkrétní adresou
./ordercli foodora reorder <orderCode> --confirm --address-id <id>

# Odhlášení
./ordercli foodora logout
```

---

## Ostatní regiony
```sh
./ordercli foodora countries
./ordercli foodora config set --country HU
./ordercli foodora config set --country SK
./ordercli foodora config set --country AT
./ordercli foodora config set --country CZ
```

---

## Import session z Chrome (bez hesla)

Pokud jsi přihlášen na foodora.cz v Chrome:
```sh
./ordercli foodora cookies chrome --url https://www.foodora.cz/ --profile "Default"
./ordercli foodora history
```

---

## ⚠️ Upozornění

Tento nástroj komunikuje s privátními API. Používej na vlastní riziko — může dojít k rate limitingu nebo zablokování.
