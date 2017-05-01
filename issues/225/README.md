```bash
sudo sysctl -w vm.max_map_count=262144

docker stack deploy -c elk.yml elk

docker stack ps elk

open "http://localhost:5601"

# Got "Unable to connect to Elasticsearch at http://localhost:9200." error message
```