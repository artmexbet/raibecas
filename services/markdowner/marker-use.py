import time

import pymupdf4llm
import ollama


def extract_with_pymupdf(pdf_path: str) -> str:
    """Извлечение текста из PDF с сохранением форматирования"""
    return pymupdf4llm.to_markdown(pdf_path)


def clean_with_ollama(text: str, model: str = "gemma3:12b") -> str:
    """
    Исправление артефактов OCR через Ollama.
    НЕ меняет содержимое, только исправляет очевидные ошибки распознавания.
    """
    prompt = f"""You are an OCR post-processor. Your task is to fix ONLY obvious OCR artifacts and errors.

STRICT RULES:
1. DO NOT change any words, sentences, or meaning
2. DO NOT add or remove any content
3. DO NOT rephrase or summarize anything
4. ONLY fix:
   - Broken words (e.g., "wo rd" -> "word")
   - Wrong characters from OCR (e.g., "rn" misread as "m", "l" as "1", "O" as "0")
   - Broken markdown formatting (e.g., incomplete headers, broken links)
5. Keep ALL original formatting, structure, headers, lists, tables
6. Return the text AS-IS if no OCR errors are found

Input text:
```
{text}
```

Output the corrected text only, no explanations:"""

    response = ollama.generate(model, prompt, options={
        "temperature": 0.0,
        "max_tokens": len(text) + 500
    })

    return response.response


def pdf_to_markdown(pdf_path: str, use_ollama_cleanup: bool = False, model: str = "gemma3:12b", verbose: bool = False) -> str:
    """
    Конвертация PDF в Markdown.

    Args:
        pdf_path: Путь к PDF файлу
        use_ollama_cleanup: Использовать ли Ollama для очистки артефактов OCR
        model: Модель Ollama для очистки
        verbose: Выводить ли отладочную информацию
    """
    start_time = time.time()
    # Шаг 1: Точное извлечение текста
    markdown = extract_with_pymupdf(pdf_path)
    print(f"Извлечение заняло {time.time() - start_time:.2f} секунд")
    if verbose:
        print(f"Длина извлеченного текста: {len(markdown)} символов")

    start_time = time.time()
    print("Начинаем очистку через Ollama..." if use_ollama_cleanup else "Очистка не требуется.")

    # Шаг 2: Опциональная очистка через Ollama
    if use_ollama_cleanup:
        # Обрабатываем по частям, если текст большой
        max_chunk_size = 4000
        if len(markdown) > max_chunk_size:
            chunks = []
            lines = markdown.split('\n')
            current_chunk = []
            current_size = 0

            for line in lines:
                if current_size + len(line) > max_chunk_size and current_chunk:
                    chunks.append('\n'.join(current_chunk))
                    current_chunk = [line]
                    current_size = len(line)
                else:
                    current_chunk.append(line)
                    current_size += len(line) + 1

            if current_chunk:
                chunks.append('\n'.join(current_chunk))

            if verbose:
                print(f"Текст разбит на {len(chunks)} частей для очистки.")

            cleaned_chunks = []
            for i, chunk in enumerate(chunks):
                if verbose:
                    print(f"Очистка части {i + 1}/{len(chunks)}...")
                cleaned_chunk = clean_with_ollama(chunk, model)
                cleaned_chunks.append(cleaned_chunk)

            markdown = '\n'.join(cleaned_chunks)
        else:
            markdown = clean_with_ollama(markdown, model)

        print(f"Очистка заняла {time.time() - start_time:.2f} секунд")

    return markdown


if __name__ == "__main__":
    # Базовое использование - только pymupdf4llm (рекомендуется)
    md = pdf_to_markdown("examples/fifth book.pdf", use_ollama_cleanup=True)

    # С очисткой через Ollama (если есть артефакты OCR)
    # md = pdf_to_markdown("examples/fifth book.pdf", use_ollama_cleanup=True, model="llama3.2")

    with open("examples/recognized.md", "w", encoding="utf-8") as f:
        f.write(md)

    print(f"Сохранено в examples/recognized.md ({len(md)} символов)")
