import urllib.request
import zipfile
import io
import json

# URL constants
url_cities = "https://download.geonames.org/export/dump/cities15000.zip"
url_admin1 = "https://download.geonames.org/export/dump/admin1CodesASCII.txt"

print("Downloading admin1 codes mapping...")
admin1_map = {}
try:
    req = urllib.request.Request(url_admin1, headers={'User-Agent': 'Mozilla/5.0'})
    with urllib.request.urlopen(req) as response:
        admin_content = response.read().decode("utf-8")
    
    for line in admin_content.splitlines():
        if not line:
            continue
        parts = line.split("\t")
        if len(parts) >= 2:
            code = parts[0] # e.g. "CA.08"
            name = parts[1] # e.g. "Ontario"
            admin1_map[code] = name
    print(f"Loaded {len(admin1_map)} admin1 region mappings.")
except Exception as e:
    print(f"Warning: Failed to load admin1 codes: {e}")

print("Downloading cities15000.zip from GeoNames...")
try:
    req = urllib.request.Request(url_cities, headers={'User-Agent': 'Mozilla/5.0'})
    with urllib.request.urlopen(req) as response:
        zip_data = response.read()
    
    print("Extracting zip...")
    with zipfile.ZipFile(io.BytesIO(zip_data)) as z:
        txt_content = z.read("cities15000.txt").decode("utf-8")
        
    print("Parsing cities15000.txt...")
    raw_cities = []
    lines = txt_content.splitlines()
    for line in lines:
        if not line:
            continue
        parts = line.split("\t")
        if len(parts) < 19:
            continue
        
        name = parts[1]
        ascii_name = parts[2]
        alt_names = [a.strip() for a in parts[3].split(",") if a.strip()]
        country = parts[8]
        admin1_code = parts[10]
        pop = int(parts[14]) if parts[14] else 0
        tz = parts[17]
        
        if not tz:
            continue
            
        # Look up region name
        lookup_key = f"{country}.{admin1_code}"
        region = admin1_map.get(lookup_key, "")
            
        raw_cities.append({
            "name": name,
            "ascii": ascii_name,
            "alt": alt_names,
            "country": country,
            "region": region,
            "pop": pop,
            "tz": tz
        })
        
    # Sort by population descending first
    raw_cities.sort(key=lambda c: c["pop"], reverse=True)
    
    # Deduplicate alternate names, giving priority to larger cities
    seen_alt = set()
    deduped_cities = []
    
    for c in raw_cities:
        filtered_alts = []
        name_lower = c["name"].lower()
        ascii_lower = c["ascii"].lower()
        
        for alt in c["alt"]:
            alt_lower = alt.lower()
            if alt_lower == name_lower or alt_lower == ascii_lower:
                continue
            if alt_lower not in seen_alt:
                filtered_alts.append(alt)
                seen_alt.add(alt_lower)
                
        seen_alt.add(name_lower)
        seen_alt.add(ascii_lower)
        
        deduped_cities.append({
            "name": c["name"],
            "ascii": c["ascii"],
            "alt": filtered_alts,
            "country": c["country"],
            "region": c["region"],
            "pop": c["pop"],
            "tz": c["tz"]
        })
        
    output_path = "cities.json"
    print(f"Saving {len(deduped_cities)} deduplicated cities with regions to {output_path}...")
    with open(output_path, "w", encoding="utf-8") as f:
        json.dump(deduped_cities, f, ensure_ascii=False)
        
    print("Done successfully!")

except Exception as e:
    print(f"Error occurred: {e}")
