import importlib.util
import pathlib
import unittest


SCRIPT_PATH = pathlib.Path(__file__).resolve().parent / 'capacity_certification.py'


spec = importlib.util.spec_from_file_location('capacity_certification', SCRIPT_PATH)
capacity_certification = importlib.util.module_from_spec(spec)
assert spec.loader is not None
spec.loader.exec_module(capacity_certification)


class CapacityCertificationTest(unittest.TestCase):
    def test_build_report_passes_checked_in_evidence(self):
        report = capacity_certification.build_report()

        self.assertEqual(report['summary']['overall_status'], 'pass')
        self.assertEqual(report['summary']['failed_lanes'], [])
        self.assertEqual(report['saturation_indicator']['status'], 'pass')
        self.assertEqual(report['mixed_workload']['status'], 'pass')
        self.assertIn('1000x24', [lane['lane'] for lane in report['soak_matrix']])


if __name__ == '__main__':
    unittest.main()
