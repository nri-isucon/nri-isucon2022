
package nriisucon.isubnb.response;

import java.util.List;
import lombok.Data;
import nriisucon.isubnb.model.Home;

@Data
public class HomesResponse {

    private int count;
    private List<Home> homes;

    public HomesResponse(List<Home> homes) {
        this.count = homes.size();
        this.homes = homes;
    }
}
