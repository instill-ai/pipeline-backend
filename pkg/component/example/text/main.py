import subprocess


def execute_go_program(file_paths, chunksize, chunk_overlap):
    file_paths_str = ",".join(file_paths)

    subprocess.run(["go", "build", "-o", "chunk_text", "main.go"])

    result = subprocess.run(
        [
            "./chunk_text",
            f"--file_paths={file_paths_str}",
            f"--chunksize={chunksize}",
            f"--chunkoverlap={chunk_overlap}",
        ],
        capture_output=True,
        text=True,
    )

    print(result.stdout)
    if result.stderr:
        print("Error:", result.stderr)

    subprocess.run(["rm", "chunk_text"])


file_paths = ["test_data_with_lists.md", "test_data_with_table_and_lists.md"]
chunksize = 800
chunk_overlap = 300

execute_go_program(file_paths, chunksize, chunk_overlap)
