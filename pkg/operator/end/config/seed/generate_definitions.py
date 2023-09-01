import jsonref
import json

from os.path import dirname


base_path = dirname(__file__)
base_uri = 'file://{}/'.format(base_path)


with open("./definitions.json") as schema_file:
    a = jsonref.loads(schema_file.read(), base_uri=base_uri,
                      jsonschema=True, merge_props=True)

with open('../definitions.json', 'w') as o:
    json.dump(a, o, indent=2)
