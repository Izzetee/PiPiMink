#!/bin/bash
# Generate Swagger documentation from annotations in the code
swag init -o docs/


# If you need to convert YAML to JSON manually
if command -v yq &> /dev/null; then
    echo "Converting YAML to JSON using yq..."
    yq -o=json eval ./docs/swagger.yaml > ./docs/swagger.json
elif command -v python3 &> /dev/null && python3 -c "import yaml" 2>/dev/null; then
    echo "Converting YAML to JSON using python..."
    python3 -c "import yaml, json, sys; yaml_file = open('./docs/swagger.yaml', 'r'); json_file = open('./docs/swagger.json', 'w'); json.dump(yaml.safe_load(yaml_file), json_file, indent=2); yaml_file.close(); json_file.close()"
else
    echo "Attempting conversion using a manual approach..."
    # Try to manually update JSON from YAML using sed/awk
    # This is a simplified approach and might not handle all YAML complexities
    # but should work for basic swagger files that don't use advanced YAML features
    
    # First, let's create a basic structure
    echo "{" > ./docs/swagger.json
    
    # Convert the YAML to a basic JSON format
    # This is a very simplified approach and might need improvements
    sed -n 's/^\([^:]*\):[[:space:]]*\(.*\)/  "\1": \2,/p' ./docs/swagger.yaml | 
      sed 's/: \([^,{[].*[^"}]\),$/: "\1",/' >> ./docs/swagger.json
    
    # Close the JSON object
    echo "}" >> ./docs/swagger.json
    
    echo "Warning: Manual conversion may be incomplete. Please install yq or python3 with pyyaml for better results."
    echo "You can install yq with: brew install yq"
    echo "You can install python dependencies with: pip install pyyaml"
    
    # Alternative option for macOS users
    if [[ "$(uname)" == "Darwin" ]]; then
        echo "For macOS users, try: brew install python && pip3 install pyyaml"
    fi
fi

echo "Swagger documentation generated successfully!"
echo "Please verify the JSON output is correctly formatted."