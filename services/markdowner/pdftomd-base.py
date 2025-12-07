import pymupdf4llm


def pdf_to_markdown(pdf_path: str, output_path: str = None) -> str:
    """Конвертация PDF в Markdown без LLM - максимальная точность"""
    md_text = pymupdf4llm.to_markdown(
        pdf_path,
        page_chunks=True,
        write_images=True,
        image_path="images",
        show_progress=True,
        margins=(0, 0, 0, 0)  # Без обрезки полей
    )

    if output_path:
        if isinstance(md_text, str):
            with open(output_path, "w", encoding="utf-8") as f:
                f.write(md_text)
        else:
            for i, chunk in enumerate(md_text):
                print(i, chunk)
                with open(f"{output_path.rsplit('.', 1)[0]}_part{i + 1}.md", "w", encoding="utf-8") as f:
                    f.write(chunk["text"])

    return md_text


if __name__ == "__main__":
    pdf_to_markdown("examples/Resume_Maydurov.pdf", "output.md")
