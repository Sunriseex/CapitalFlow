#!/bin/bash

# Интерактивный скрипт добавления платежей
# Работает с обновленной модульной структурой payments-manager

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CONFIG_DIR="$HOME/.config/finance"

# Цвета для вывода
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log() {
    echo -e "${GREEN}[$(date +'%Y-%m-%d %H:%M:%S')]${NC} $1"
}

error() {
    echo -e "${RED}[ERROR]${NC} $1" >&2
}

show_categories() {
    echo "Доступные категории:"
    echo "1. subscriptions - Подписки"
    echo "2. utilities - Коммунальные услуги"
    echo "3. hosting - Хостинг"
    echo "4. food - Еда"
    echo "5. rent - Аренда"
    echo "6. transport - Транспорт"
    echo "7. entertainment - Развлечения"
    echo "8. healthcare - Здоровье"
    echo "9. other - Другое"
}

show_accounts() {
    echo "Доступные счета:"
    echo "1. b:Yandex"
    echo "2. b:Tinkoff"
    echo "3. b:AlfaBank"
    echo "4. a:Cash"
    echo "5. a:TinkoffCard"
}

get_category() {
    local category_num=$1
    case $category_num in
    1) echo "subscriptions" ;;
    2) echo "utilities" ;;
    3) echo "hosting" ;;
    4) echo "food" ;;
    5) echo "rent" ;;
    6) echo "transport" ;;
    7) echo "entertainment" ;;
    8) echo "healthcare" ;;
    9) echo "other" ;;
    *) echo "subscriptions" ;;
    esac
}

get_account() {
    local account_num=$1
    case $account_num in
    1) echo "b:Yandex" ;;
    2) echo "b:Tinkoff" ;;
    3) echo "b:AlfaBank" ;;
    4) echo "a:Cash" ;;
    5) echo "a:TinkoffCard" ;;
    *) echo "b:Yandex" ;;
    esac
}

validate_date() {
    local date_str=$1
    if [[ ! $date_str =~ ^[0-9]{4}-[0-9]{2}-[0-9]{2}$ ]]; then
        return 1
    fi
    date -d "$date_str" >/dev/null 2>&1
    return $?
}

validate_amount() {
    local amount=$1
    if [[ ! $amount =~ ^[0-9]+(\.[0-9]{1,2})?$ ]]; then
        return 1
    fi
    return 0
}

main() {
    echo "=========================================="
    echo "Добавление нового платежа"
    echo "=========================================="

    # Название платежа
    while true; do
        read -p "Введите название платежа: " name
        if [[ -n "$name" ]]; then
            break
        else
            error "Название не может быть пустым"
        fi
    done

    # Сумма платежа
    while true; do
        read -p "Введите сумму в рублях (например: 349.90): " amount
        if validate_amount "$amount"; then
            break
        else
            error "Некорректная сумма. Пример: 250.50 или 1000"
        fi
    done

    # Выбор типа ввода даты
    echo ""
    echo "Выберите способ указания срока платежа:"
    echo "1) Указать конкретную дату (ГГГГ-ММ-ДД)"
    echo "2) Указать количество дней от текущей даты"

    while true; do
        read -p "Ваш выбор [1 или 2]: " choice
        case $choice in
        1)
            while true; do
                read -p "Введите дату (ГГГГ-ММ-ДД): " date_input
                if validate_date "$date_input"; then
                    date_param="--date"
                    date_value="$date_input"
                    days_param=""
                    break
                else
                    error "Некорректная дата. Формат: ГГГГ-ММ-ДД"
                fi
            done
            break
            ;;
        2)
            while true; do
                read -p "Введите количество дней: " days_input
                if [[ $days_input =~ ^[0-9]+$ ]] && [ $days_input -gt 0 ]; then
                    days_param="--days"
                    days_value="$days_input"
                    date_param=""
                    break
                else
                    error "Введите положительное число"
                fi
            done
            break
            ;;
        *)
            error "Введите 1 или 2"
            ;;
        esac
    done

    # Тип платежа
    echo ""
    echo "Типы платежей:"
    echo "1) monthly - Ежемесячный"
    echo "2) yearly - Ежегодный"
    echo "3) one-time - Разовый"

    while true; do
        read -p "Выберите тип платежа [1-3, по умолчанию 1]: " type_choice
        type_choice=${type_choice:-1}
        case $type_choice in
        1)
            payment_type="monthly"
            break
            ;;
        2)
            payment_type="yearly"
            break
            ;;
        3)
            payment_type="one-time"
            break
            ;;
        *) error "Введите число от 1 до 3" ;;
        esac
    done

    # Категория
    echo ""
    show_categories
    while true; do
        read -p "Выберите категорию [1-9, по умолчанию 1]: " category_choice
        category_choice=${category_choice:-1}
        if [[ $category_choice =~ ^[1-9]$ ]]; then
            category=$(get_category $category_choice)
            break
        else
            error "Введите число от 1 до 9"
        fi
    done

    # Счет для ledger
    echo ""
    show_accounts
    while true; do
        read -p "Выберите счет [1-5, по умолчанию 1]: " account_choice
        account_choice=${account_choice:-1}
        if [[ $account_choice =~ ^[1-5]$ ]]; then
            ledger_account=$(get_account $account_choice)
            break
        else
            error "Введите число от 1 до 5"
        fi
    done

    # Подтверждение
    echo ""
    echo "=========================================="
    echo "Проверьте введенные данные:"
    echo "=========================================="
    echo "Название: $name"
    echo "Сумма: $amount руб."
    if [[ -n "$date_value" ]]; then
        echo "Дата: $date_value"
    else
        echo "Через дней: $days_value"
    fi
    echo "Тип: $payment_type"
    echo "Категория: $category"
    echo "Счет: $ledger_account"
    echo "=========================================="

    while true; do
        read -p "Добавить платеж? [y/N]: " confirm
        case $confirm in
        [Yy]*) break ;;
        [Nn]*)
            echo "Отмена добавления платежа"
            exit 0
            ;;
        *)
            echo "Отмена добавления платежа"
            exit 0
            ;;
        esac
    done

    # Выполнение команды
    if [[ -n "$date_value" ]]; then
        payments-manager add --name "$name" --amount "$amount" --date "$date_value" --type "$payment_type" --category "$category" --ledger-account "$ledger_account"
    else
        payments-manager add --name "$name" --amount "$amount" --days "$days_value" --type "$payment_type" --category "$category" --ledger-account "$ledger_account"
    fi

    if [ $? -eq 0 ]; then
        log "Платеж успешно добавлен!"
    else
        error "Ошибка при добавлении платежа"
    fi
}

# Проверка зависимостей
check_dependencies() {
    if ! command -v payments-manager &>/dev/null; then
        error "payments-manager не найден. Убедитесь, что он установлен и доступен в PATH"
        exit 1
    fi
}

check_dependencies
main "$@"
