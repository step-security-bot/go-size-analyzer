from argparse import ArgumentParser

import requests

from define import IntegrationTest, TestType
from gsa import build_gsa
from merge import merge_covdata
from remote import load_remote_binaries, load_remote_for_tui_test
from utils import *


def eval_test(gsa: str, target: IntegrationTest):
    name = target.name
    path = target.path
    test_type = target.type

    if TestType.TEXT_TEST in test_type:
        run_process([gsa, "-f", "text", "--verbose", path], name, ".txt")

    if TestType.JSON_TEST in test_type:
        run_process([gsa,
                     "-f", "json",
                     "--indent", "2",
                     path,
                     "-o", get_result_file(name, ".json")],
                    name, ".json.txt")

    if TestType.HTML_TEST in test_type:
        run_process([gsa,
                     "-f", "html",
                     path,
                     "-o", get_result_file(name, ".html")],
                    name, ".html.txt")

    if TestType.SVG_TEST in test_type:
        run_process([gsa,
                     "-f", "svg",
                     path,
                     "-o", get_result_file(name, ".svg")],
                    name, ".svg.txt")


def run_unit_tests():
    log("Running unit tests...")
    load_remote_for_tui_test()

    unit_path = os.path.join(get_project_root(), "covdata", "unit")

    run_process(
        [
            "go",
            "test",
            "-v",
            "-race",
            "-covermode=atomic",
            "-cover",
            "-tags=embed",
            "./...",
            f"-test.gocoverdir={unit_path}"
        ],
        "unit_embed",
        ".txt",
        timeout=600,  # Windows runner is extremely slow
    )

    # test no tag
    run_process(
        [
            "go",
            "test",
            "-v",
            "-race",
            "-covermode=atomic",
            "-cover",
            "./internal/webui",
            f"-test.gocoverdir={unit_path}",
        ],
        "unit",
        ".txt",
        timeout=600,  # Windows runner is extremely slow
    )

    log("Unit tests passed.")


def run_integration_tests():
    log("Running integration tests...")

    targets = load_remote_binaries()

    with build_gsa() as gsa:
        run_web_test(gsa)

        all_tests = len(targets)
        completed_tests = 1

        for target in targets:
            base = time.time()
            try:
                eval_test(gsa, target)
                log(f"[{completed_tests}/{all_tests}] Test {target.name} passed in {format_time(time.time() - base)}.")
                completed_tests += 1
            except Exception as e:
                log(f"Test {target.name} failed: {e}")
                raise e

    log("Integration tests passed.")


def run_web_test(entry: str):
    log("Running web test...")

    env = os.environ.copy()
    env["GOCOVERDIR"] = get_covdata_integration_dir()

    port = find_unused_port()
    if port is None:
        raise Exception("Failed to find an unused port.")

    p = subprocess.Popen(
        args=[entry, "--web", "--listen", f"127.0.0.1:{port}", entry],
        text=True, cwd=get_project_root(),
        encoding="utf-8", env=env, stdout=subprocess.PIPE, stderr=subprocess.PIPE
    )

    for line in iter(p.stdout.readline, ""):
        if "localhost" in line:
            break

    time.sleep(1)  # still need to wait for the server to start

    ret = requests.get(f"http://localhost:{port}").text

    assert_html_valid(ret)

    p.terminate()
    log("Web test passed.")


def get_parser() -> ArgumentParser:
    parser = ArgumentParser()

    parser.add_argument("--unit", action="store_true", help="Run unit tests.")
    parser.add_argument("--integration", action="store_true", help="Run integration tests.")

    return parser


if __name__ == "__main__":
    parser = get_parser()
    args = parser.parse_args()

    init_dirs()

    if not args.unit and not args.integration:
        if os.getenv("CI") is not None:
            args.unit = True
            args.integration = True
        else:
            raise Exception("Please specify a test type to run.")

    if args.unit:
        run_unit_tests()
    if args.integration:
        run_integration_tests()

    merge_covdata()

    log("All tests passed.")
