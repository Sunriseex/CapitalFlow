#!/usr/bin/env bash

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CONFIG_DIR="$HOME/.config/waybar"
DEPOSITS_FILE="$CONFIG_DIR/deposits.json"
LEDGER_PATH="$HOME/ObsidianVault/finances/transactions.ledger"

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

warn() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

check_dependencies() {
    if ! command -v deposit-manager &>/dev/null; then
        error "deposit-manager не найден. Убедитесь, что он установлен и доступен в PATH"
        exit 1
    fi

    if ! command -v jq &>/dev/null; then
        error "Утилита jq не найдена. Установите её в ваш configuration.nix"
        exit 1
    fi
}

show_existing_deposits() {
    echo "=========================================="
    echo "Ваши вклады:"
    echo "=========================================="

    if [[ -f "$DEPOSITS_FILE" ]]; then
        timeout 10s deposit-manager list
        local exit_code=$?

        if [[ $exit_code -eq 124 ]]; then
            error "Команда выполняется слишком долго, прерываю..."
            return 1
        elif [[ $exit_code -ne 0 ]]; then
            error "Ошибка при выполнении команды (код: $exit_code)"
            return 1
        fi
    else
        echo "Файл вкладов не найден: $DEPOSITS_FILE"
    fi
}

calculate_days_until_date() {
    local target_date="$1"
    local current_date=$(date +%Y-%m-%d)

    local target_sec=$(date -d "$target_date" +%s 2>/dev/null)
    local current_sec=$(date -d "$current_date" +%s 2>/dev/null)

    if [[ -z "$target_sec" || -z "$current_sec" ]]; then
        echo "0"
        return 1
    fi

    local diff_sec=$((target_sec - current_sec))
    local days=$((diff_sec / 86400))

    if [[ $days -lt 0 ]]; then
        echo "0"
    else
        echo "$days"
    fi
}

extract_income_from_output() {
    local output="$1"
    echo "$output" | grep "Ожидаемый доход" | sed -E 's/.* ([0-9]+\.[0-9]+) руб.*/\1/' | head -1
}

calculate_single_deposit_income() {
    echo "=========================================="
    echo "Расчет дохода по вкладу до даты"
    echo "=========================================="

    show_existing_deposits
    echo ""

    local deposits_json
    if [[ -f "$DEPOSITS_FILE" ]]; then
        deposits_json=$(jq -r '.deposits[] | "\(.name)|\(.bank)|\(.id)"' "$DEPOSITS_FILE")
    else
        error "Файл вкладов не найден"
        return 1
    fi

    if [[ -z "$deposits_json" ]]; then
        error "Нет доступных вкладов"
        return 1
    fi

    local i=1
    local deposits=()
    while IFS= read -r line; do
        if [[ -n "$line" ]]; then
            deposits[i]="$line"
            local name=$(echo "$line" | cut -d'|' -f1)
            local bank=$(echo "$line" | cut -d'|' -f2)
            echo "  $i) $name ($bank)"
            ((i++))
        fi
    done <<<"$deposits_json"

    while true; do
        read -p "Выберите вклад [1-$((i - 1))]: " choice
        if [[ $choice =~ ^[0-9]+$ ]] && [[ $choice -ge 1 ]] && [[ $choice -le $((i - 1)) ]]; then
            selected="${deposits[$choice]}"
            deposit_name=$(echo "$selected" | cut -d'|' -f1)
            deposit_bank=$(echo "$selected" | cut -d'|' -f2)
            deposit_id=$(echo "$selected" | cut -d'|' -f3)
            break
        else
            error "Введите число от 1 до $((i - 1))"
        fi
    done

    while true; do
        read -p "Введите дату окончания расчета (ГГГГ-ММ-ДД): " target_date
        if date -d "$target_date" >/dev/null 2>&1; then
            days=$(calculate_days_until_date "$target_date")
            if [[ $days -gt 0 ]]; then
                break
            else
                error "Дата должна быть в будущем"
            fi
        else
            error "Некорректная дата. Используйте формат ГГГГ-ММ-ДД"
        fi
    done

    log "Расчет дохода по вкладу '$deposit_name' за $days дней..."

    local output
    output=$(deposit-manager calculate "$deposit_id" "$days" 2>&1)
    echo "$output"
}

days_between() {
    local start_date="$1"
    local end_date="$2"

    local start_sec=$(date -d "$start_date" +%s 2>/dev/null)
    local end_sec=$(date -d "$end_date" +%s 2>/dev/null)

    if [[ -z "$start_sec" || -z "$end_sec" ]]; then
        echo "0"
        return 1
    fi

    local diff_sec=$((end_sec - start_sec))
    local days=$((diff_sec / 86400))
    echo "$days"
}

calculate_total_term_income() {
    echo "=========================================="
    echo "Расчет дохода за весь срок по срочным вкладам"
    echo "=========================================="

    # Получаем список вкладов
    if [[ ! -f "$DEPOSITS_FILE" ]]; then
        error "Файл вкладов не найден"
        return 1
    fi

    local deposits_json
    deposits_json=$(jq -r '.deposits[] | "\(.name)|\(.bank)|\(.id)|\(.type)|\(.end_date)|\(.start_date)"' "$DEPOSITS_FILE")

    if [[ -z "$deposits_json" ]]; then
        error "Нет доступных вкладов"
        return 1
    fi

    local total_income_full_term=0
    local total_income_remaining=0
    local deposit_count=0

    echo ""
    echo "Доход по срочным вкладам:"
    echo "----------------------------------------"

    while IFS= read -r line; do
        if [[ -n "$line" ]]; then
            deposit_name=$(echo "$line" | cut -d'|' -f1)
            deposit_bank=$(echo "$line" | cut -d'|' -f2)
            deposit_id=$(echo "$line" | cut -d'|' -f3)
            deposit_type=$(echo "$line" | cut -d'|' -f4)
            deposit_end_date=$(echo "$line" | cut -d'|' -f5)
            deposit_start_date=$(echo "$line" | cut -d'|' -f6)

            # Показываем только срочные вклады с указанными датами
            if [[ "$deposit_type" == "term" && "$deposit_end_date" != "null" && -n "$deposit_end_date" && "$deposit_start_date" != "null" && -n "$deposit_start_date" ]]; then
                total_days=$(days_between "$deposit_start_date" "$deposit_end_date")
                remaining_days=$(calculate_days_until_date "$deposit_end_date")

                if [[ $total_days -gt 0 ]]; then
                    echo "  $deposit_name ($deposit_bank):"
                    echo "    Период: с $deposit_start_date по $deposit_end_date"
                    echo "    Всего дней: $total_days, осталось: $remaining_days"

                    # Расчет дохода за весь срок
                    local output_full
                    output_full=$(deposit-manager calculate "$deposit_id" "$total_days" 2>&1)
                    local result_full
                    result_full=$(extract_income_from_output "$output_full")

                    if [[ -n "$result_full" && "$result_full" =~ ^[0-9]+\.?[0-9]*$ ]]; then
                        echo "    Доход за весь срок: $result_full руб."
                        total_income_full_term=$(echo "$total_income_full_term + $result_full" | bc -l 2>/dev/null || echo "0")
                    else
                        echo "    Доход за весь срок: ошибка расчета"
                    fi

                    # Расчет дохода за оставшийся срок
                    if [[ $remaining_days -gt 0 ]]; then
                        local output_remaining
                        output_remaining=$(deposit-manager calculate "$deposit_id" "$remaining_days" 2>&1)
                        local result_remaining
                        result_remaining=$(extract_income_from_output "$output_remaining")

                        if [[ -n "$result_remaining" && "$result_remaining" =~ ^[0-9]+\.?[0-9]*$ ]]; then
                            echo "    Доход за оставшийся срок: $result_remaining руб."
                            total_income_remaining=$(echo "$total_income_remaining + $result_remaining" | bc -l 2>/dev/null || echo "0")
                        else
                            echo "    Доход за оставшийся срок: ошибка расчета"
                        fi
                    else
                        echo "    Срок вклада истек"
                    fi

                    echo ""
                    ((deposit_count++))
                else
                    echo "  $deposit_name ($deposit_bank): некорректный срок"
                fi
            fi
        fi
    done <<<"$deposits_json"

    echo "----------------------------------------"
    if [[ $deposit_count -gt 0 ]]; then
        printf "ИТОГО: %d срочных вкладов\n" "$deposit_count"
        printf "Общий доход за весь срок: %.2f руб.\n" "$total_income_full_term"
        if (($(echo "$total_income_remaining > 0" | bc -l))); then
            printf "Общий доход за оставшийся срок: %.2f руб.\n" "$total_income_remaining"
        fi
    else
        echo "Нет активных срочных вкладов"
    fi
}

calculate_all_deposits_income() {
    echo "=========================================="
    echo "Расчет общего дохода по всем вкладам до даты"
    echo "=========================================="

    while true; do
        read -p "Введите дату окончания расчета (ГГГГ-ММ-ДД): " target_date
        if date -d "$target_date" >/dev/null 2>&1; then
            days=$(calculate_days_until_date "$target_date")
            if [[ $days -gt 0 ]]; then
                break
            else
                error "Дата должна быть в будущем"
            fi
        else
            error "Некорректная дата. Используйте формат ГГГГ-ММ-ДД"
        fi
    done

    if [[ ! -f "$DEPOSITS_FILE" ]]; then
        error "Файл вкладов не найден"
        return 1
    fi

    local deposits_json
    deposits_json=$(jq -r '.deposits[] | "\(.name)|\(.bank)|\(.id)"' "$DEPOSITS_FILE")

    if [[ -z "$deposits_json" ]]; then
        error "Нет доступных вкладов"
        return 1
    fi

    local total_income=0
    local deposit_count=0

    echo ""
    echo "Расчет дохода по каждому вкладу:"
    echo "----------------------------------------"

    while IFS= read -r line; do
        if [[ -n "$line" ]]; then
            deposit_name=$(echo "$line" | cut -d'|' -f1)
            deposit_bank=$(echo "$line" | cut -d'|' -f2)
            deposit_id=$(echo "$line" | cut -d'|' -f3)

            echo -n "  $deposit_name ($deposit_bank): "

            local output
            output=$(deposit-manager calculate "$deposit_id" "$days" 2>&1)
            local result
            result=$(extract_income_from_output "$output")

            if [[ -n "$result" && "$result" =~ ^[0-9]+\.?[0-9]*$ ]]; then
                echo "$result руб."
                total_income=$(echo "$total_income + $result" | bc -l 2>/dev/null || echo "0")
                ((deposit_count++))
            else
                echo "ошибка расчета"
            fi
        fi
    done <<<"$deposits_json"

    echo "----------------------------------------"
    if [[ $deposit_count -gt 0 ]]; then
        printf "ИТОГО: %d вкладов, общий доход: %.2f руб.\n" "$deposit_count" "$total_income"
    else
        echo "Не удалось рассчитать доход по вкладам"
    fi
}

create_new_deposit_interactive() {
    echo "=========================================="
    echo "Создание нового вклада"
    echo "=========================================="

    while true; do
        read -p "Введите название вклада: " name
        if [[ -n "$name" ]]; then
            break
        else
            error "Название не может быть пустым"
        fi
    done

    bank="Yandex"
    echo "Банк: $bank (автоматически)"

    echo ""
    echo "Типы вкладов:"
    echo "1) savings - Бессрочный (с возможностью пополнения)"
    echo "2) term - Срочный (фиксированный срок)"

    while true; do
        read -p "Выберите тип вклада [1 или 2]: " type_choice
        case $type_choice in
        1)
            deposit_type="savings"
            break
            ;;
        2)
            deposit_type="term"
            break
            ;;
        *)
            error "Введите 1 или 2"
            ;;
        esac
    done

    while true; do
        read -p "Введите сумму вклада в рублях: " amount
        if [[ $amount =~ ^[0-9]+(\.[0-9]{1,2})?$ ]] && (($(echo "$amount > 0" | bc -l))); then
            break
        else
            error "Некорректная сумма. Пример: 50000 или 1500.50"
        fi
    done

    while true; do
        read -p "Введите процентную ставку (например: 17.5): " rate
        if [[ $rate =~ ^[0-9]+(\.[0-9]{1,2})?$ ]] && (($(echo "$rate > 0" | bc -l))); then
            break
        else
            error "Некорректная ставка. Пример: 17.5 или 8.25"
        fi
    done

    local term=""
    if [[ "$deposit_type" == "term" ]]; then
        echo ""
        echo "Срок вклада:"
        echo "1) 3 месяца (91 день)"
        echo "2) 6 месяцев (181 день)"
        echo "3) 1 год (367 дней)"
        echo "4) 2 года (730 дней)"

        while true; do
            read -p "Выберите срок вклада [1-4]: " term_choice
            case $term_choice in
            1)
                term=3
                break
                ;;
            2)
                term=6
                break
                ;;
            3)
                term=12
                break
                ;;
            4)
                term=24
                break
                ;;
            *) error "Введите число от 1 до 4" ;;
            esac
        done
    fi

    local command="deposit-manager create --name \"$name\" --bank \"$bank\" --type \"$deposit_type\" --amount \"$amount\" --rate \"$rate\""

    if [[ "$deposit_type" == "term" && -n "$term" ]]; then
        command="$command --term \"$term\""
    fi

    echo ""
    echo "Выполняется команда: $command"
    eval "$command"
}

topup_deposit_interactive() {
    echo "=========================================="
    echo "Пополнение вклада"
    echo "=========================================="

    show_existing_deposits
    echo ""

    local deposits_json
    if [[ -f "$DEPOSITS_FILE" ]]; then
        deposits_json=$(jq -r '.deposits[] | "\(.name)|\(.bank)|\(.id)"' "$DEPOSITS_FILE")
    else
        error "Файл вкладов не найден"
        return 1
    fi

    if [[ -z "$deposits_json" ]]; then
        error "Нет доступных вкладов"
        return 1
    fi

    local i=1
    local deposits=()
    while IFS= read -r line; do
        if [[ -n "$line" ]]; then
            deposits[i]="$line"
            local name=$(echo "$line" | cut -d'|' -f1)
            local bank=$(echo "$line" | cut -d'|' -f2)
            echo "  $i) $name ($bank)"
            ((i++))
        fi
    done <<<"$deposits_json"

    while true; do
        read -p "Выберите вклад для пополнения [1-$((i - 1))]: " choice
        if [[ $choice =~ ^[0-9]+$ ]] && [[ $choice -ge 1 ]] && [[ $choice -le $((i - 1)) ]]; then
            selected="${deposits[$choice]}"
            deposit_name=$(echo "$selected" | cut -d'|' -f1)
            deposit_bank=$(echo "$selected" | cut -d'|' -f2)
            deposit_id=$(echo "$selected" | cut -d'|' -f3)
            break
        else
            error "Введите число от 1 до $((i - 1))"
        fi
    done

    while true; do
        read -p "Введите сумму для пополнения: " amount
        if [[ $amount =~ ^[0-9]+(\.[0-9]{1,2})?$ ]] && (($(echo "$amount > 0" | bc -l))); then
            break
        else
            error "Некорректная сумма. Пример: 50000 или 1500.50"
        fi
    done

    log "Пополнение вклада '$deposit_name' на $amount руб."
    deposit-manager topup "$deposit_id" "$amount"
}

update_deposit_rate_interactive() {
    echo "=========================================="
    echo "Обновление ставки по вкладу"
    echo "=========================================="

    echo "Список ваших вкладов:"
    local deposits_list
    deposits_list=$(deposit-manager list 2>/dev/null)

    if [[ -z "$deposits_list" ]]; then
        error "Нет доступных вкладов для обновления"
        return 1
    fi

    echo "$deposits_list"
    echo ""

    while true; do
        read -p "Введите название вклада: " name
        if [[ -n "$name" ]]; then
            break
        else
            error "Название не может быть пустым"
        fi
    done

    while true; do
        read -p "Введите банк: " bank
        if [[ -n "$bank" ]]; then
            break
        else
            error "Банк не может быть пустым"
        fi
    done

    log "Поиск вклада '$name' в банке '$bank'..."

    # Получаем информацию о вкладе из файла
    if [[ ! -f "$DEPOSITS_FILE" ]]; then
        error "Файл вкладов не найден: $DEPOSITS_FILE"
        return 1
    fi

    local deposit_json
    deposit_json=$(jq -r --arg name "$name" --arg bank "$bank" '.deposits[] | select(.name == $name and .bank == $bank)' "$DEPOSITS_FILE" 2>/dev/null)

    if [[ -z "$deposit_json" || "$deposit_json" == "null" ]]; then
        error "Вклад '$name' в банке '$bank' не найден"
        return 1
    fi

    local current_rate
    current_rate=$(echo "$deposit_json" | jq -r '.interest_rate')
    local current_promo_rate
    current_promo_rate=$(echo "$deposit_json" | jq -r '.promo_rate // "нет"')
    local current_promo_end_date
    current_promo_end_date=$(echo "$deposit_json" | jq -r '.promo_end_date // "нет"')

    echo ""
    echo "Текущие данные вклада:"
    echo "  Основная ставка: $current_rate%"
    echo "  Промо-ставка: $current_promo_rate"
    if [[ "$current_promo_end_date" != "нет" ]]; then
        echo "  Дата окончания промо: $current_promo_end_date"
    fi
    echo ""

    while true; do
        read -p "Введите новую основную ставку (текущая: $current_rate%): " new_rate
        if [[ -z "$new_rate" ]]; then
            new_rate="$current_rate"
            break
        elif [[ $new_rate =~ ^[0-9]+(\.[0-9]{1,2})?$ ]] && (($(echo "$new_rate > 0" | bc -l))); then
            break
        else
            error "Некорректная ставка. Пример: 5.5 или 10"
        fi
    done

    echo ""
    echo "Обновление промо-ставки:"
    echo "1) Оставить текущую промо-ставку"
    echo "2) Изменить промо-ставку"
    echo "3) Удалить промо-ставку"

    local promo_choice
    while true; do
        read -p "Ваш выбор [1-3]: " promo_choice
        case $promo_choice in
        1)
            new_promo_rate="$current_promo_rate"
            new_promo_end_date="$current_promo_end_date"
            break
            ;;
        2)
            while true; do
                read -p "Введите новую промо-ставку: " new_promo_rate
                if [[ $new_promo_rate =~ ^[0-9]+(\.[0-9]{1,2})?$ ]] && (($(echo "$new_promo_rate > 0" | bc -l))); then
                    break
                else
                    error "Некорректная промо-ставка. Пример: 5.5 или 10"
                fi
            done

            while true; do
                read -p "Введите дату окончания промо-ставки (ГГГГ-ММ-ДД): " new_promo_end_date
                if date -d "$new_promo_end_date" >/dev/null 2>&1; then
                    break
                else
                    error "Некорректная дата. Используйте формат ГГГГ-ММ-ДД"
                fi
            done
            break
            ;;
        3)
            new_promo_rate="null"
            new_promo_end_date="null"
            break
            ;;
        *)
            error "Введите 1, 2 или 3"
            ;;
        esac
    done

    echo ""
    echo "Подтвердите изменения:"
    echo "  Новая основная ставка: $new_rate%"
    if [[ "$new_promo_rate" != "null" ]]; then
        echo "  Новая промо-ставка: $new_promo_rate%"
        echo "  Дата окончания промо: $new_promo_end_date"
    else
        echo "  Промо-ставка: будет удалена"
    fi
    echo ""

    while true; do
        read -p "Подтвердить обновление? [y/N]: " confirm
        case $confirm in
        [Yy]*)
            break
            ;;
        [Nn]*)
            echo "Отмена обновления"
            return 0
            ;;
        *)
            echo "Отмена обновления"
            return 0
            ;;
        esac
    done

    log "Обновление ставки вклада '$name'..."

    local temp_file
    temp_file=$(mktemp)

    if [[ "$new_promo_rate" == "null" ]]; then
        jq --arg name "$name" \
            --arg bank "$bank" \
            --arg new_rate "$new_rate" \
            '(.deposits[] | select(.name == $name and .bank == $bank)) |= 
            (.interest_rate = ($new_rate | tonumber) | del(.promo_rate) | del(.promo_end_date))' \
            "$DEPOSITS_FILE" >"$temp_file"
    else
        jq --arg name "$name" \
            --arg bank "$bank" \
            --arg new_rate "$new_rate" \
            --arg new_promo_rate "$new_promo_rate" \
            --arg new_promo_end_date "$new_promo_end_date" \
            '(.deposits[] | select(.name == $name and .bank == $bank)) |= 
            (.interest_rate = ($new_rate | tonumber) | .promo_rate = ($new_promo_rate | tonumber) | .promo_end_date = $new_promo_end_date)' \
            "$DEPOSITS_FILE" >"$temp_file"
    fi

    if [[ $? -eq 0 ]]; then
        mv "$temp_file" "$DEPOSITS_FILE"
        log "Ставка по вкладу '$name' успешно обновлена"
        echo "✅ Ставка по вкладу успешно обновлена"
    else
        error "Ошибка при обновлении ставки"
        rm -f "$temp_file"
        return 1
    fi
}

main_menu() {
    echo "=========================================="
    echo "Управление вкладами - deposit-manager"
    echo "=========================================="
    echo ""
    echo "Выберите действие:"
    echo "1) Создать новый вклад"
    echo "2) Просмотреть список вкладов"
    echo "3) Рассчитать доход по выбранному вкладу до даты"
    echo "4) Рассчитать общий доход по всем вкладам до даты"
    echo "5) Рассчитать доход за весь срок по срочным вкладам"
    echo "6) Пополнить существующий вклад"
    echo "7) Обновить ставку по вкладу"
    echo "0) Выход"
    echo ""
}

main() {
    check_dependencies

    while true; do
        main_menu
        read -p "Ваш выбор [0-9]: " choice

        case $choice in
        1)
            create_new_deposit_interactive
            ;;
        2)
            echo ""
            show_existing_deposits
            ;;
        3)
            calculate_single_deposit_income
            ;;
        4)
            calculate_all_deposits_income
            ;;
        5)
            calculate_total_term_income
            ;;
        6)
            topup_deposit_interactive
            ;;
        7)
            update_deposit_rate_interactive
            ;;
        0)
            echo "Выход..."
            exit 0
            ;;
        *)
            error "Неверный выбор"
            ;;
        esac

        echo ""
        read -p "Нажмите Enter для продолжения..."
    done
}

case "${1:-}" in
"create")
    create_new_deposit_interactive
    ;;
"list")
    show_existing_deposits
    ;;
"calculate-single")
    calculate_single_deposit_income
    ;;
"calculate-all")
    calculate_all_deposits_income
    ;;
"calculate-term")
    calculate_total_term_income
    ;;
"topup")
    topup_deposit_interactive
    ;;
"update-rate")
    update_deposit_rate_interactive
    ;;
"help" | "-h" | "--help")
    echo "Использование: $0 [command]"
    echo ""
    echo "Команды:"
    echo "  create           - Создать новый вклад"
    echo "  list             - Показать список вкладов"
    echo "  calculate-single - Рассчитать доход по вкладу до даты"
    echo "  calculate-all    - Рассчитать общий доход по всем вкладам"
    echo "  calculate-term   - Рассчитать доход за весь срок по срочным вкладам"
    echo "  topup            - Пополнить существующий вклад"
    echo "  update-rate      - Обновить ставку по вкладу"
    echo "  help             - Показать эту справку"
    echo ""
    echo "Без аргументов - интерактивный режим"
    ;;
*)
    if [[ -z "$1" ]]; then
        main
    else
        error "Неизвестная команда: $1"
        echo "Используйте '$0 help' для справки"
        exit 1
    fi
    ;;
esac
