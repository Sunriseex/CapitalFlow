#!/usr/bin/env bash

RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m'

log() {
    echo -e "${GREEN}[$(date +'%Y-%m-%d %H:%M:%S')]${NC} $1"
}

error() {
    echo -e "${RED}[ERROR]${NC} $1" >&2
}

info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

info "Запуск автоматического начисления процентов"

if command -v deposit-manager &>/dev/null; then
    if deposit-manager accrue-interest; then
        info "Начисление процентов завершено успешно"
    else
        error "Ошибка при начислении процентов (код: $?)"
        exit 1
    fi
else
    error "deposit-manager не найден в PATH"
    exit 1
fi
